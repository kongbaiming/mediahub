import http.server, socketserver, threading, os, sys

os.chdir(r"D:\project\mediahub\docs\prototype")
PORT = 8765
Handler = http.server.SimpleHTTPRequestHandler
Handler.extensions_map.update({".html": "text/html; charset=utf-8"})

httpd = socketserver.TCPServer(("127.0.0.1", PORT), Handler)
print(f"Serving on http://localhost:{PORT}/", flush=True)

# Run in a separate thread so we can shut down cleanly
def serve():
    httpd.serve_forever()

t = threading.Thread(target=serve, daemon=True)
t.start()

# Wait for shutdown signal
try:
    t.join()
except KeyboardInterrupt:
    httpd.shutdown()