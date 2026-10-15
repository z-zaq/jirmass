"""
Gunicorn configuration for Jirmass Django application.
"""
import os

# Worker temporary directory in memory to prevent worker heartbeats from hanging
worker_tmp_dir = "/dev/shm"

# Disable or reroute Gunicorn 25.1+ control server
# Prevents permission errors attempting to bind gunicorn.ctl in /nonexistent or $HOME
control_socket_disable = True
control_socket = os.environ.get("GUNICORN_CONTROL_SOCKET", "/tmp/gunicorn.ctl")
