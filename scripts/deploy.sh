#!/bin/sh
# Build the production images, put them on the server, and check the result.
#
# This is the deploy that has been done by hand so far, written down. It runs
# from a checkout on the machine that has the Docker daemon used for building;
# the server is reached over SSH and only receives finished images and files.
#
#   DEPLOY_HOST=root@192.168.1.2 ./scripts/deploy.sh frontend
#   DEPLOY_HOST=root@192.168.1.2 ./scripts/deploy.sh backend
#   DEPLOY_HOST=root@192.168.1.2 ./scripts/deploy.sh nginx
#
# Each part is independent, and each ends with a check that would have caught
# the failures already seen here: the frontend check keeps the addresses the
# router knows at 200 and a wrong one at 404, and the backend check proves the
# gallery limiter refuses a token after its window instead of trusting that the
# route it is mounted on is ever reached.
#
# What it does not do: change database migrations, touch .env, or force a
# container when the tag has not changed. Those stay manual on purpose.
set -eu

PART=${1:-}
DEPLOY_HOST=${DEPLOY_HOST:-}
STACK=${STACK:-/opt/compose/web-photography}
SITE=${SITE:-https://elvove.site}
TAG=${TAG:-prod-$(date +%Y.%m.%d%H%M)-go}

REPO=$(cd "$(dirname "$0")/.." && pwd)   # the repository root: deploy.sh lives in scripts/

die() {
	echo "deploy: $*" >&2
	exit 1
}

[ -n "$DEPLOY_HOST" ] || die "set DEPLOY_HOST, e.g. DEPLOY_HOST=root@192.168.1.2"
[ -n "$PART" ] || die "say what to deploy: frontend, backend or nginx"

remote() {
	ssh "$DEPLOY_HOST" "$@"
}

# check_url <path> <expected status>
check_url() {
	got=$(curl -s -o /dev/null -w '%{http_code}' "$SITE$1")
	[ "$got" = "$2" ] || die "$1 answered $got, expected $2"
	echo "  ok  $1 $got"
}

deploy_frontend() {
	echo "== frontend: build"
	(cd "$REPO/frontend" && npm run build >/dev/null)
	[ -f "$REPO/frontend/nginx.prod.conf" ] || die "missing frontend/nginx.prod.conf"
	[ -f "$REPO/frontend/Dockerfile.prod" ] || die "missing frontend/Dockerfile.prod"

	echo "== frontend: send dist, Dockerfile and nginx config"
	tar -C "$REPO/frontend" -czf /tmp/deploy-dist.tar.gz dist
	scp -q /tmp/deploy-dist.tar.gz "$DEPLOY_HOST:$STACK/_build/frontend/dist.tar.gz"
	scp -q "$REPO/frontend/nginx.prod.conf" "$DEPLOY_HOST:$STACK/_build/frontend/nginx.prod.conf"
	scp -q "$REPO/frontend/Dockerfile.prod" "$DEPLOY_HOST:$STACK/_build/frontend/Dockerfile"
	rm -f /tmp/deploy-dist.tar.gz

	echo "== frontend: build the image on the server"
	remote "set -e
		cd $STACK/_build/frontend
		rm -rf dist && mkdir dist && tar -C dist -xzf dist.tar.gz --strip-components=1 && rm dist.tar.gz
		docker build -q -t elvove-frontend:$TAG . >/dev/null
		echo built elvove-frontend:$TAG"

	echo "== frontend: point compose.yml at it, restart, check"
	remote "set -e
		cd $STACK
		cp compose.yml compose.yml.prev-\$(date +%Y.%m.%d-%H%M)
		sed -i 's|image: elvove-frontend:[^ ]*|image: elvove-frontend:$TAG|' compose.yml
		docker compose up -d frontend 2>&1 | tail -1"
	sleep 4
	check_url / 200
	check_url /about 200
	check_url /admin 200
	check_url /halo 404
}

deploy_backend() {
	echo "== backend: test, build"
	(cd "$REPO/backend-go" && go vet ./... && go test ./... >/dev/null)
	(cd "$REPO/backend-go" && CGO_ENABLED=0 GOOS=linux GOARCH=arm64 \
		go build -a -trimpath -ldflags="-s -w" -o /tmp/deploy-api ./cmd/api)

	echo "== backend: send the binary and build the image"
	scp -q /tmp/deploy-api "$DEPLOY_HOST:$STACK/_build/backend/elvove-api.new"
	rm -f /tmp/deploy-api
	remote "set -e
		cd $STACK/_build/backend
		cp elvove-api elvove-api.prev-\$(date +%Y%m%d%H%M)
		mv elvove-api.new elvove-api
		docker build -q -t elvove-backend:$TAG . >/dev/null
		echo built elvove-backend:$TAG"

	echo "== backend: point compose.yml at it, restart, wait for healthy"
	remote "set -e
		cd $STACK
		cp compose.yml compose.yml.prev-\$(date +%Y.%m.%d-%H%M)
		sed -i 's|image: elvove-backend:[^ ]*|image: elvove-backend:$TAG|' compose.yml
		docker compose up -d backend 2>&1 | tail -1
		for i in \$(seq 1 40); do
			[ \"\$(docker inspect -f '{{.State.Health.Status}}' photography-backend)\" = healthy ] && exit 0
			sleep 1
		done
		echo 'backend did not become healthy' >&2; exit 1"

	echo "== backend: the limiter must refuse the 31st request of one token"
	refused=$(remote "for i in \$(seq 1 35); do
			curl -s -o /dev/null -w '%{http_code}\n' http://127.0.0.1:8080/api/gallery/cccccccccccccccccccccccccccccccc/photos
		done" | grep -c '^429$' || true)
	[ "$refused" -gt 0 ] || die "no 429 in 35 requests: the gallery limiter is not reached"
	echo "  ok  gallery limiter refused $refused of 35"
	check_url /api/public/packages 200
}

deploy_nginx() {
	echo "== nginx: the main config and default.conf go in by bind mount"
	scp -q "$REPO/nginx/nginx.conf" "$DEPLOY_HOST:$STACK/nginx/nginx.conf"
	scp -q "$REPO/nginx/default.conf" "$DEPLOY_HOST:$STACK/nginx/default.conf"

	echo "== nginx: compose.yml must mount both and raise nofile"
	remote "set -e
		cd $STACK
		grep -q './nginx/nginx.conf:/etc/nginx/nginx.conf:ro' compose.yml || {
			echo 'compose.yml does not mount nginx.conf; see the repository copy' >&2; exit 1; }
		grep -q 'nofile' compose.yml || {
			echo 'compose.yml has no nofile ulimit' >&2; exit 1; }"

	echo "== nginx: recreate and check what the container actually got"
	remote "set -e
		cd $STACK
		docker compose up -d --force-recreate nginx 2>&1 | tail -1"
	sleep 4
	remote "docker exec photography-nginx nginx -t 2>&1 | tail -1"
	limit=$(remote "docker exec photography-nginx sh -c 'ulimit -n'")
	[ "$limit" = "65535" ] || die "container nofile is $limit, expected 65535"
	echo "  ok  ulimit -n $limit"
	remote "docker exec photography-nginx sh -c 'grep -q \"worker_connections  16384\" /etc/nginx/nginx.conf'" ||
		die "the container is not running the repository's nginx.conf"
	echo "  ok  worker_connections 16384"
	check_url / 200
	check_url /api/public/packages 200
}

case "$PART" in
frontend) deploy_frontend ;;
backend) deploy_backend ;;
nginx) deploy_nginx ;;
*) die "unknown part '$PART': use frontend, backend or nginx" ;;
esac

echo "deploy: $PART done"
