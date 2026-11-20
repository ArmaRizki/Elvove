#!/bin/sh
# Check the parts of HSTS that can be checked from outside: is HTTPS the only
# way in, and is the header there yet?
#
#   ./scripts/check-hsts.sh [site]        # site defaults to https://elvove.site
#
# HSTS is set in Cloudflare's dashboard, not in this repository's nginx: the
# tunnel terminates TLS at Cloudflare, so a header added here would be one more
# copy behind the one that actually reaches the browser. This script only looks
# at what the browser sees, and says which step is missing.
#
# Exit status: 0 everything ready (or the header already sent), 1 something to
# fix — the message says what.
set -eu

SITE=${1:-${SITE:-https://elvove.site}}
SITE=${SITE%/}
HOST=${SITE#*://}
HOST=${HOST%%/*}
MAX_AGE=${HSTS_MAX_AGE:-15552000}

problem=0
say() { printf '%s\n' "$*"; }

say "checking $HOST"

# 1. Plain HTTP must lead somewhere on HTTPS. HSTS starts with one visit over
#    HTTPS; a visitor who never gets there never receives the header, and an
#    address that answers 200 on HTTP means the cleartext copy is a real site.
plain=$(curl -s -o /dev/null -w '%{http_code} %{redirect_url}' "http://$HOST/" || true)
plain_code=${plain%% *}
plain_to=${plain#* }
case "$plain_code" in
200)
	say "  FAIL http://$HOST answers 200 over cleartext instead of redirecting to HTTPS"
	say "       Cloudflare: SSL/TLS → Edge Certificates → Always Use HTTPS = On"
	say "       (this is also what HSTS needs: the first visit has to arrive on HTTPS)"
	problem=1
	;;
301 | 302 | 307 | 308)
	case "$plain_to" in
	https://*)
		say "  ok   http://$HOST redirects ($plain_code) to $plain_to"
		;;
	*)
		say "  FAIL http://$HOST redirects to $plain_to, which is not HTTPS"
		problem=1
		;;
	esac
	;;
esac

# 2. The site is reachable over HTTPS at all, before anything is promised. When
#    the address given is already http, step 1 answered this: following the
#    redirect from here would only fail on a target that cannot be reached from
#    the machine running the check.
case "$SITE" in
http://*)
	say "  note checked over http only; pass https://… for the header itself"
	;;
*)
	code=$(curl -s -o /dev/null -w '%{http_code}' "$SITE/")
	if [ "$code" = "200" ]; then
		say "  ok   $SITE answers 200"
	else
		say "  FAIL $SITE answers $code"
		problem=1
	fi
	;;
esac

# 3. The header. Type first, then the parts that matter.
header=$(curl -s -D - -o /dev/null "$SITE/" | grep -i '^strict-transport-security:' || true)
if [ -z "$header" ]; then
	say "  none Strict-Transport-Security is not sent yet"
	say "       Cloudflare: SSL/TLS → Edge Certificates → HTTP Strict Transport Security"
	say "       → Enable HSTS, max-age 15552000 (180 days), include subdomains"
	say "       only if every subdomain is HTTPS, preload only once the rest is settled."
	problem=1
else
	say "  ok   $header"
	age=$(printf '%s' "$header" | tr ';' '\n' | sed -n 's/.*max-age=\([0-9]*\).*/\1/p' | head -1)
	if [ -n "$age" ] && [ "$age" -lt "$MAX_AGE" ]; then
		say "  warn max-age=$age is shorter than $MAX_AGE; browsers forget the promise sooner"
	fi
	printf '%s' "$header" | grep -qi 'includeSubDomains' &&
		say "  note includeSubDomains is on: every subdomain must stay HTTPS" ||
		say "  note includeSubDomains is off"
	printf '%s' "$header" | grep -qi 'preload' &&
		say "  note preload is on: removing it later is slow (the list is cached by browsers)"
fi

# 4. A redirect is the normal, finished state: it is what "Always Use HTTPS"
#    does, and once HSTS is on the browser does not even ask. So it is reported
#    as such instead of counted as a failure.
case "${plain_code:-}" in
301 | 302 | 307 | 308) plain_ok=1 ;;
*) plain_ok=0 ;;
esac

say ""
if [ "$problem" -eq 0 ]; then
	say "ready: HTTPS is the only way in and the header is sent"
elif [ "$plain_ok" -eq 1 ]; then
	say "HTTP already redirects to HTTPS. Left: turn on HSTS in Cloudflare (the 'none' line above)."
else
	say "not ready: see the FAIL lines above; they name the switch to turn on"
fi
exit "$problem"
