#!/bin/sh
set -e

# If a custom command is provided (e.g., docker run ... bash or python manage.py ...), execute it directly
if [ "$#" -gt 0 ] && [ "$1" != "gunicorn" ] && [ "$1" != "entrypoint.sh" ] && [ "$1" != "/usr/local/bin/entrypoint.sh" ] && [ "$1" != "/app/entrypoint.sh" ]; then
    exec "$@"
fi

# Remove entrypoint/gunicorn command wrapper argument if passed
if [ "$1" = "gunicorn" ] || [ "$1" = "entrypoint.sh" ] || [ "$1" = "/usr/local/bin/entrypoint.sh" ] || [ "$1" = "/app/entrypoint.sh" ]; then
    shift
fi

# If the next argument is the WSGI target (e.g. config.wsgi:application), shift it to prevent duplication
if [ "$1" = "config.wsgi:application" ] || [ "$1" = "config.wsgi" ]; then
    shift
fi

echo "==> Running Django database migrations..."
python manage.py migrate --noinput

echo "==> Seeding initial water products..."
python manage.py seed_water_products

echo "==> Collecting static files..."
python manage.py collectstatic --noinput

# Determine if Gunicorn supports control socket flags (introduced in Gunicorn v25.1.0)
# Disable unneeded control socket or reroute it to /tmp to prevent binding in HOME or /nonexistent
EXTRA_ARGS=""
if gunicorn --help 2>&1 | grep -q -- '--no-control-socket'; then
    EXTRA_ARGS="--no-control-socket"
elif gunicorn --help 2>&1 | grep -q -- '--control-socket'; then
    EXTRA_ARGS="--control-socket /tmp/gunicorn.ctl"
fi

echo "==> Starting Gunicorn on port ${PORT:-8000}..."
exec gunicorn config.wsgi:application \
    --bind "0.0.0.0:${PORT:-8000}" \
    --workers 3 \
    --timeout 60 \
    --worker-tmp-dir /dev/shm \
    $EXTRA_ARGS \
    "$@"

