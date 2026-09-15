import argparse
import base64
import datetime
import json
import os
import socket
import struct


def read_exact(connection, size):
    data = bytearray()
    while len(data) < size:
        chunk = connection.recv(size - len(data))
        if not chunk:
            raise RuntimeError("Gateway disconnected before sending a complete response")
        data.extend(chunk)
    return bytes(data)


def read_reply(reader):
    line = reader.readline()
    if not line.endswith(b"\r\n"):
        raise RuntimeError("Incomplete Valkey reply")
    kind, value = line[:1], line[1:-2]
    if kind == b"-":
        raise RuntimeError(value.decode())
    if kind == b"+":
        return value
    if kind == b":":
        return int(value)
    if kind == b"$":
        size = int(value)
        if size == -1:
            return None
        data = reader.read(size)
        if len(data) != size or reader.read(2) != b"\r\n":
            raise RuntimeError("Incomplete Valkey bulk reply")
        return data
    if kind == b"*":
        size = int(value)
        if size == -1:
            return None
        return [read_reply(reader) for _ in range(size)]
    raise RuntimeError("Unexpected Valkey reply type")


def execute(connection, reader, *arguments):
    parts = [str(value).encode() for value in arguments]
    data = b"*" + str(len(parts)).encode() + b"\r\n"
    for part in parts:
        data += b"$" + str(len(part)).encode() + b"\r\n" + part + b"\r\n"
    connection.sendall(data)
    return read_reply(reader)


parser = argparse.ArgumentParser(description="Send one EGTS packet and check both example Valkey streams")
parser.add_argument("--host", default="127.0.0.1")
parser.add_argument("--port", type=int, default=28085)
parser.add_argument("--stream", default="egts.packets")
parser.add_argument("--database", type=int, default=0)
args = parser.parse_args()

raw = bytes.fromhex(
    "0100000b002300000001991800000001ef0000000202101500"
    "d2312b104fba3a9ed227bc35030000b200000000006a8d"
)
started = datetime.datetime.now(datetime.timezone.utc)

with socket.create_connection((args.host, args.port), timeout=30) as connection:
    connection.sendall(raw)
    header = read_exact(connection, 10)
    header_size = header[3]
    if header_size not in (11, 16):
        raise RuntimeError("Invalid response header length")
    header += read_exact(connection, header_size - len(header))
    body_size = struct.unpack_from("<H", header, 5)[0]
    if header[9] != 0 or body_size < 3:
        raise RuntimeError("Expected an EGTS_PT_RESPONSE packet")
    body = read_exact(connection, body_size + 2)
    packet_id, result = struct.unpack_from("<HB", body)
    if packet_id != 0 or result != 0:
        raise RuntimeError(f"Packet was not accepted: RPID={packet_id}, PR={result}")
    print(f"Gateway ACK: RPID={packet_id}, PR={result}")

for name, default_port in (("monitoring", 6379), ("backup", 6380)):
    suffix = name.upper()
    host = os.getenv(f"VALKEY_HOST_{suffix}", "127.0.0.1")
    port = int(os.getenv(f"VALKEY_PORT_{suffix}", str(default_port)))
    username = os.getenv(f"VALKEY_USERNAME_{suffix}", "egts_backup" if name == "backup" else "")
    password = os.getenv(f"VALKEY_PASSWORD_{suffix}", f"egts_{name}")
    with socket.create_connection((host, port), timeout=10) as connection:
        with connection.makefile("rb") as reader:
            if password:
                auth = ["AUTH"]
                if username:
                    auth.append(username)
                auth.append(password)
                execute(connection, reader, *auth)
            if args.database:
                execute(connection, reader, "SELECT", args.database)
            entries = execute(connection, reader, "XREVRANGE", args.stream, "+", "-", "COUNT", 100)
    for entry_id, fields in entries:
        data = dict(zip(fields[::2], fields[1::2]))
        if b"payload" not in data:
            continue
        event = json.loads(data[b"payload"])
        received = datetime.datetime.fromisoformat(event["record"]["received_at"].replace("Z", "+00:00"))
        if received >= started and base64.b64decode(event["record"]["raw"], validate=True) == raw:
            print(f"{name}: original packet received ({len(raw)} bytes), stream={args.stream}, id={entry_id.decode()}")
            break
    else:
        raise RuntimeError(f"No matching fresh packet among the last 100 entries in {name}")
