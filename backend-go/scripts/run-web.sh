#!/usr/bin/env bash
# Runs the whole site with the Go backend instead of the Java one.
#
# The Go server only exposes the API and /uploads; it does not serve the SPA. So
# this builds the frontend to static files, puts nginx in front of both, and
# points every /api request at the Go container. It mirrors the production
# topology (nginx -> SPA + API) rather than the Vite dev server.
#
# The Java stack keeps running untouched: the Go containers only read the
# existing database and are named with the "elvove-go-" prefix. The Java backend
# still owns port 8080.
#
# Usage:  ./scripts/run-web.sh [--stop] [--copy-db]
#   --stop      remove the Go containers and exit
#   --copy-db   copy the database into a throwaway container and point the Go
#               backend at the copy, so writes from the site never touch the
#               Java database. Recommended when clicking around.
#
# Then open http://localhost:8091
set -euo pipefail

WEB_PORT=${WEB_PORT:-8091}
DB_CONTAINER=${DB_CONTAINER:-photography-local-db}
JAVA_CONTAINER=${JAVA_CONTAINER:-photography-local-backend}
DB_NAME=${DB_NAME:-photographer_portfolio}
API_CONTAINER=elvove-go-dev
WEB_CONTAINER=elvove-go-web
COPY_DB_CONTAINER=elvove-go-copy-db
IMAGE=elvove-backend-go:verify
COPY_DB=0

for arg in "$@"; do
  case $arg in
    --stop) stop_requested=1 ;;
    --copy-db) COPY_DB=1 ;;
    *) echo "error: unknown option $arg" >&2; exit 1 ;;
  esac
done

REPO_ROOT=$(cd "$(dirname "$0")/../.." && pwd)
BACKEND_DIR="$REPO_ROOT/backend-go"
FRONTEND_DIR="$REPO_ROOT/frontend"
# local-data/ is gitignored, so build output never lands in the repository.
WORK_DIR="$REPO_ROOT/local-data/go-web"

stop() {
  docker rm -f "$WEB_CONTAINER" "$API_CONTAINER" >/dev/null 2>&1 || true
}

if [[ -n ${stop_requested:-} ]]; then
  stop
  docker rm -f "$COPY_DB_CONTAINER" >/dev/null 2>&1 || true
  echo "Go web containers removed."
  exit 0
fi

docker inspect "$DB_CONTAINER" >/dev/null 2>&1 || {
  echo "error: $DB_CONTAINER is not running; start the Java stack first" >&2
  exit 1
}

NETWORK=$(docker inspect "$DB_CONTAINER" --format '{{range $k,$_ := .NetworkSettings.Networks}}{{$k}}{{end}}')

# Read the database password and the mail key from the running Java containers
# instead of asking for them, so no secret is written to disk here.
DB_PASSWORD=$(docker exec "$DB_CONTAINER" printenv POSTGRES_PASSWORD)
JWT_SECRET=$(docker exec "$JAVA_CONTAINER" printenv JWT_SECRET 2>/dev/null || true)
SMTP_KEY=$(docker exec "$JAVA_CONTAINER" printenv SMTP_ENCRYPTION_KEY 2>/dev/null || true)
# A Go JWT is only valid within this process, so a dev-only secret is harmless.
[[ ${#JWT_SECRET} -ge 32 ]] || JWT_SECRET='dev-only-secret-at-least-32-bytes!!'

mkdir -p "$WORK_DIR"

echo '1/4 building the Go image'
docker build -q -t "$IMAGE" "$BACKEND_DIR" >/dev/null

echo '2/4 building the frontend'
if [[ ! -d "$WORK_DIR/src/node_modules" ]]; then
  mkdir -p "$WORK_DIR/src"
  tar -c -C "$FRONTEND_DIR" --exclude=node_modules --exclude=dist . | tar -x -C "$WORK_DIR/src"
  (cd "$WORK_DIR/src" && npm ci --legacy-peer-deps --no-audit --no-fund >/dev/null)
fi
# An empty base URL keeps requests same-origin, so nginx decides where /api goes.
(cd "$WORK_DIR/src" && VITE_API_BASE_URL= npm run build >/dev/null)
rm -rf "$WORK_DIR/dist" && cp -r "$WORK_DIR/src/dist" "$WORK_DIR/dist"

echo '3/4 copying uploads out of the Java container'
if [[ ! -d "$WORK_DIR/uploads" ]]; then
  mkdir -p "$WORK_DIR/uploads"
  docker cp "$JAVA_CONTAINER:/app/uploads/." "$WORK_DIR/uploads/" 2>/dev/null || true
fi

echo '4/4 starting the containers'
stop

# --copy-db runs the site against a snapshot, so clicking through the admin or
# sending a test booking can never change the Java database.
if [[ $COPY_DB -eq 1 ]]; then
  docker rm -f "$COPY_DB_CONTAINER" >/dev/null 2>&1 || true
  docker run -d --name "$COPY_DB_CONTAINER" --network "$NETWORK" \
    -e POSTGRES_PASSWORD=copy -e "POSTGRES_DB=$DB_NAME" postgres:16-alpine >/dev/null
  for _ in $(seq 1 30); do
    docker exec "$COPY_DB_CONTAINER" pg_isready -U postgres -d "$DB_NAME" >/dev/null 2>&1 && break
    sleep 1
  done
  docker exec "$DB_CONTAINER" pg_dump -U postgres --no-owner --no-privileges "$DB_NAME" \
    | docker exec -i "$COPY_DB_CONTAINER" psql -U postgres -d "$DB_NAME" -q >/dev/null
  DB_CONTAINER=$COPY_DB_CONTAINER
  DB_PASSWORD=copy
  echo '   using a snapshot in '"$COPY_DB_CONTAINER"' (Java database is read-only)'
  # The snapshot carries the Java admin hash, which nobody knows. Set the same
  # dev password on every rebuild so it does not change between restarts.
  if "$BACKEND_DIR/scripts/set-dev-admin.sh" >/dev/null 2>&1; then
    echo '   dev admin password set (see: scripts/set-dev-admin.sh --show)'
  else
    echo '   warning: could not set the dev admin password' >&2
  fi
fi

cat > "$WORK_DIR/nginx.conf" <<'CONF'
server {
    listen 80;
    server_name _;
    server_tokens off;

    add_header X-Content-Type-Options "nosniff" always;
    add_header X-Frame-Options "DENY" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;

    client_max_body_size 100M;

    # "goapi" is the Go container's network alias, so this file stays portable.
    location /api/ {
        proxy_pass http://goapi:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $http_x_forwarded_proto;
    }

    location /uploads/ {
        proxy_pass http://goapi:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-Proto $http_x_forwarded_proto;
    }

    location / {
        root /usr/share/nginx/html;
        index index.html;
        # SPA: unknown paths fall back to index.html for React Router.
        try_files $uri $uri/ /index.html;
    }
}
CONF

docker run -d --name "$API_CONTAINER" \
  --network "$NETWORK" --network-alias goapi \
  -e "DB_URL=postgres://postgres:$(printf '%s' "$DB_PASSWORD" | sed 's/@/%40/g')@$DB_CONTAINER:5432/$DB_NAME?sslmode=disable" \
  -e "JWT_SECRET=$JWT_SECRET" \
  -e "SMTP_ENCRYPTION_KEY=$SMTP_KEY" \
  -e PORT=8080 -e UPLOAD_DIR=/app/uploads \
  -e "FRONTEND_URL=http://localhost:$WEB_PORT" \
  -e "MAIL_SITE_URL=http://localhost:$WEB_PORT" \
  -v "$WORK_DIR/uploads:/app/uploads" \
  "$IMAGE" >/dev/null

docker run -d --name "$WEB_CONTAINER" \
  --network "$NETWORK" \
  -p "127.0.0.1:$WEB_PORT:80" \
  -v "$WORK_DIR/nginx.conf:/etc/nginx/conf.d/default.conf:ro" \
  -v "$WORK_DIR/dist:/usr/share/nginx/html:ro" \
  nginx:alpine >/dev/null

for _ in $(seq 1 30); do
  curl -sf "http://127.0.0.1:$WEB_PORT/api/public/about" >/dev/null 2>&1 && break
  sleep 1
done

if curl -sf "http://127.0.0.1:$WEB_PORT/api/public/about" >/dev/null; then
  printf '\nGo backend + site ready: http://localhost:%s\n' "$WEB_PORT"
  printf 'Java backend is untouched on http://localhost:8080\n'
  printf 'Stop with: %s --stop\n' "$0"
else
  echo "error: the site did not come up; check: docker logs $API_CONTAINER" >&2
  exit 1
fi
