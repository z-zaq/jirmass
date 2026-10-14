from http.server import BaseHTTPRequestHandler
import os
import sys
from pathlib import Path

# Add project root to sys.path so it can find dev_server
root_dir = Path(__file__).resolve().parent.parent
if str(root_dir) not in sys.path:
    sys.path.insert(0, str(root_dir))

from dev_server import JirmassHTTPHandler

class handler(JirmassHTTPHandler):
    """
    Vercel Serverless Function entrypoint for Jirmass frontend.
    Handles HTML templates, static files, direct order submissions, and chatbot.
    """
    pass
