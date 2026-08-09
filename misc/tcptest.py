#!/usr/bin/env python3

import socket
import threading
import time


# =========================
# Configuration
# =========================

SERVER_HOST = "127.0.0.1"
SERVER_PORT = 9908

# "sequential" or "parallel"
MODE = "sequential"

# Time to keep connections open after the last payload was sent
RESPONSE_WAIT_SECONDS = 5

CLIENTS = [
	{
		"name": "Test",
		"payload_hex": "00",
		"keep_connected": True,
	},
]


# =========================
# Runtime state
# =========================

active_sockets = {}

payloads_sent = 0
payloads_lock = threading.Lock()

all_payloads_sent = threading.Event()
shutdown_event = threading.Event()

socket_lock = threading.Lock()


# =========================
# Helpers
# =========================

def parse_hex_payload(hex_string: str) -> bytes:
	return bytes.fromhex(hex_string)


def log(client_name: str, message: str):
	print(f"[{client_name}] {message}")


def mark_payload_sent():
	global payloads_sent

	with payloads_lock:
		payloads_sent += 1

		if payloads_sent == len(CLIENTS):
			all_payloads_sent.set()


# =========================
# TCP handling
# =========================

def receive_loop(client_name: str, sock: socket.socket):
	try:
		while not shutdown_event.is_set():
			try:
				data = sock.recv(4096)

				if not data:
					log(client_name, "Server closed connection")
					break

				log(
					client_name,
					f"Received {len(data)} bytes: "
					f"{data.hex(' ').upper()}",
				)

			except socket.timeout:
				continue

	except OSError:
		# Socket closed during shutdown
		pass

	except Exception as e:
		if not shutdown_event.is_set():
			log(client_name, f"Receive error: {e}")


def client_worker(client_config: dict):
	name = client_config["name"]
	payload = parse_hex_payload(client_config["payload_hex"])
	keep_connected = client_config["keep_connected"]

	sock = socket.socket(
		socket.AF_INET,
		socket.SOCK_STREAM,
	)

	sock.settimeout(1.0)

	try:
		log(name, f"Connecting to {SERVER_HOST}:{SERVER_PORT}")

		sock.connect(
			(
				SERVER_HOST,
				SERVER_PORT,
			)
		)

		log(name, "Connected")

		with socket_lock:
			active_sockets[name] = sock

		receiver = threading.Thread(
			target=receive_loop,
			args=(name, sock),
			daemon=True,
		)

		receiver.start()

		sock.sendall(payload)

		log(name, f"Sent {len(payload)} bytes")

		mark_payload_sent()

		if keep_connected:
			log(name, "Keeping connection open")

			while not shutdown_event.is_set():
				shutdown_event.wait(0.5)

		else:
			log(name, "Closing connection")

	except Exception as e:
		log(name, f"Error: {e}")

		# Still count this client as finished so the test can terminate
		mark_payload_sent()

	finally:
		with socket_lock:
			active_sockets.pop(name, None)

		try:
			sock.close()
		except Exception:
			pass

		log(name, "Disconnected")


# =========================
# Execution modes
# =========================

def run_parallel():
	threads = []

	for client in CLIENTS:
		thread = threading.Thread(
			target=client_worker,
			args=(client,),
		)

		thread.start()
		threads.append(thread)

	wait_for_completion()

	for thread in threads:
		thread.join()


def run_sequential():
	threads = []

	for client in CLIENTS:
		thread = threading.Thread(
			target=client_worker,
			args=(client,),
		)

		thread.start()
		threads.append(thread)

		if not client["keep_connected"]:
			thread.join()

	wait_for_completion()

	for thread in threads:
		thread.join()


# =========================
# Completion / shutdown
# =========================

def wait_for_completion():
	log("SYSTEM", "Waiting for all payloads to be sent")

	all_payloads_sent.wait()

	log(
		"SYSTEM",
		f"All payloads sent, waiting "
		f"{RESPONSE_WAIT_SECONDS}s for responses",
	)

	time.sleep(RESPONSE_WAIT_SECONDS)

	shutdown()


def shutdown():
	if shutdown_event.is_set():
		return

	shutdown_event.set()

	log("SYSTEM", "Closing remaining sockets")

	with socket_lock:
		sockets = list(active_sockets.items())

	for name, sock in sockets:
		try:
			log(name, "Closing socket")
			sock.close()
		except Exception:
			pass

	log("SYSTEM", "Shutdown complete")


# =========================
# Main
# =========================

def main():
	try:
		if MODE == "parallel":
			run_parallel()

		elif MODE == "sequential":
			run_sequential()

		else:
			raise ValueError(
				"MODE must be 'parallel' or 'sequential'"
			)

	except KeyboardInterrupt:
		shutdown()


if __name__ == "__main__":
	main()