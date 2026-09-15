import argparse
import base64
import datetime
import json
import os
import socket
import struct
import urllib.parse
import urllib.request


def read_exact(connection, size):
    data = bytearray()
    while len(data) < size:
        chunk = connection.recv(size - len(data))
        if not chunk:
            raise RuntimeError("Gateway disconnected before sending a complete response")
        data.extend(chunk)
    return bytes(data)


parser = argparse.ArgumentParser(description="Send one EGTS packet and check both example RabbitMQ queues")
parser.add_argument("--host", default="127.0.0.1")
parser.add_argument("--port", type=int, default=28084)
parser.add_argument("--queue", default="egts.packets")
parser.add_argument("--vhost", default="egts")
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

for name, default_port in (("monitoring", 15672), ("backup", 15673)):
    suffix = name.upper()
    host = os.getenv(f"RABBITMQ_HOST_{suffix}", "127.0.0.1")
    port = int(os.getenv(f"RABBITMQ_MANAGEMENT_PORT_{suffix}", str(default_port)))
    user = os.getenv(f"RABBITMQ_USER_{suffix}", f"egts_{name}")
    password = os.getenv(f"RABBITMQ_PASSWORD_{suffix}", f"egts_{name}")
    vhost = urllib.parse.quote(args.vhost, safe="")
    queue = urllib.parse.quote(args.queue, safe="")
    url = f"http://{host}:{port}/api/queues/{vhost}/{queue}/get"
    # Leave checked messages in the queue for inspection in the management UI.
    payload = json.dumps({"count": 100, "ackmode": "ack_requeue_true", "encoding": "auto"}).encode()
    request = urllib.request.Request(url, data=payload, headers={"Content-Type": "application/json"})
    token = base64.b64encode(f"{user}:{password}".encode()).decode()
    request.add_header("Authorization", f"Basic {token}")
    with urllib.request.urlopen(request, timeout=10) as response:
        messages = json.load(response)
    for message in messages:
        payload = message["payload"]
        if message["payload_encoding"] == "base64":
            payload = base64.b64decode(payload, validate=True)
        event = json.loads(payload)
        received = datetime.datetime.fromisoformat(event["record"]["received_at"].replace("Z", "+00:00"))
        if received >= started and base64.b64decode(event["record"]["raw"], validate=True) == raw:
            print(f"{name}: original packet received ({len(raw)} bytes), queue={args.queue}")
            break
    else:
        raise RuntimeError(f"No matching fresh packet among the first 100 messages in {name}; check consumers and queue backlog")
