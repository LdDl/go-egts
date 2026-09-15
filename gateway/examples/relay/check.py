import base64
import datetime
import json
import pathlib
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


raw = bytes.fromhex(
    "0100000b002300000001991800000001ef0000000202101500"
    "d2312b104fba3a9ed227bc35030000b200000000006a8d"
)

started = datetime.datetime.now(datetime.timezone.utc)

with socket.create_connection(("127.0.0.1", 28081), timeout=30) as connection:
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

for name in ("monitoring", "backup"):
    path = pathlib.Path("data/relay-demo") / name / "packets/packets.ndjson"
    lines = path.read_text().splitlines()
    if not lines:
        raise RuntimeError(f"No packets in {path}")
    event = json.loads(lines[-1])
    received = datetime.datetime.fromisoformat(event["record"]["received_at"].replace("Z", "+00:00"))
    if received < started:
        raise RuntimeError(f"The last packet in {path} was received before this check")
    actual = base64.b64decode(event["record"]["raw"], validate=True)
    if actual != raw:
        raise RuntimeError(f"The last packet in {path} differs from the sent packet")
    print(f"{name}: original packet received ({len(raw)} bytes), {path}")
