#!/bin/sh
set -e

# If a custom command is provided (e.g., docker run ... bash or python manage.py ...), execute it directly
if [ "$#" -gt 0 ] && [ "$1" != "gunicorn" ] && [ "$1" != "entrypoint.sh" ] && [ "$1" != "/usr/local/bin/entrypoint.sh" ] && [ "$1" != "/app/entrypoint.sh" ]; then
    exec "$@"
fi

# Strip leading command/wrapper arguments if passed (e.g. from CMD, ENTRYPOINT, or dockerCommand)
while [ "$#" -gt 0 ]; do
    case "$1" in
        gunicorn|entrypoint.sh|/usr/local/bin/entrypoint.sh|/app/entrypoint.sh|config.wsgi:application|config.wsgi)
            shift
            ;;
        --no-control-socket)
            shift
            ;;
        *)
            break
            ;;
    esac
done

echo "==> Running Django database migrations..."
python manage.py migrate --noinput

echo "==> Seeding initial water products..."
python manage.py seed_water_products

echo "==> Collecting static files..."
python manage.py collectstatic --noinput

# Determine if custom bind was passed in remaining arguments to avoid duplicate port binding
BIND_ARG="--bind 0.0.0.0:${PORT:-8000}"
for arg in "$@"; do
    case "$arg" in
        --bind|-b|--bind=*)
            BIND_ARG=""
            break
            ;;
    esac
done

echo "==> Starting Gunicorn on port ${PORT:-8000}..."
exec gunicorn config.wsgi:application \
    $BIND_ARG \
    --workers 3 \
    --timeout 60 \
    --worker-tmp-dir /dev/shm \
    --no-control-socket \
    "$@"


