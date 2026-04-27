#!/bin/sh

PASS=0
FAIL=0
SOCKET=/run/xdprobe/xdprobe.sock

ok() {
    printf "\033[32mOK\033[0m\n"
    PASS=$((PASS + 1))
}

ko() {
    printf "\033[31mKO\033[0m %s\n" "$1"
    FAIL=$((FAIL + 1))
    exit 1
}

check() {
    printf "%-60s " "$1"
}

check "Unix socket exists at $SOCKET"
if [ -S "$SOCKET" ]; then ok; else ko "socket not found"; fi

check "Retrieve API password from env"
# PASSWORD=$(sudo grep "PASSWORD=" /etc/sysconfig/xdprobe 2>/dev/null | awk -F "=" '{print $2}' || echo "unknown")
if [ -n "$PASSWORD" ] && [ "$PASSWORD" != "unknown" ]; then ok; else ko "password not found"; fi

get() {
    endpoint=$1
    expected=$2
    code=$(curl -X GET --unix-socket "$SOCKET" -o /dev/null -s -w "%{http_code}" -b cookies.txt "http://localhost${endpoint}")
    if [ "$code" -eq "$expected" ]; then ok; else ko "got $code"; fi
}

get_data() {
    endpoint=$1
    expected=$2
    data=$(curl -X GET --unix-socket "$SOCKET" -s -b cookies.txt "http://localhost${endpoint}")
    if [ "$data" = "$expected" ]; then ok; else ko "got $data instead of $expected"; fi
}

post_form() {
    data=$1
    endpoint=$2
    expected=$3
    code=$(curl -X POST --unix-socket "$SOCKET" -o /dev/null -s -w "%{http_code}" -d "$data" -c cookies.txt -H "Content-Type: application/x-www-form-urlencoded" "http://localhost${endpoint}")
    if [ "$code" -eq "$expected" ]; then ok; else ko "got $code"; fi
}

post_json() {
    data=$1
    endpoint=$2
    expected=$3
    code=$(curl -X POST --unix-socket "$SOCKET" -o /dev/null -s -w "%{http_code}" -d "$data" -b cookies.txt -H "Content-Type: application/json" "http://localhost${endpoint}")
    if [ "$code" -eq "$expected" ]; then ok; else ko "got $code"; fi
}

delete_policy_if_exists() {
    ip=$1
    code=$(curl -X DELETE --unix-socket "$SOCKET" -o /dev/null -s -w "%{http_code}" -b cookies.txt "http://localhost/policy/${ip}")
    if [ "$code" -eq "204" ] || [ "$code" -eq "404" ]; then ok; else ko "got $code"; fi
}

sse() {
    endpoint=$1
    curl --unix-socket "$SOCKET" -o /dev/null -s -b cookies.txt -H "Accept: text/event-stream" -N --max-time 3 "http://localhost${endpoint}"
    rc="$?"
    # 28 = timeout (connection stayed open), 0 = server closed cleanly — both are success
    if [ "$rc" -eq "28" ]; then ok; else ko "got $rc"; fi
}

check "Root endpoint returns 303"
get "/" 303

check "/auth/login/ endpoint returns 200"
get "/auth/login/" 200

check "/auth/login/ with form data returns 303"
post_form "username=admin&password=$PASSWORD" "/auth/login/" 303

check "Root endpoint with cookie returns 200"
get "/" 200

check "SSE endpoint returns data"
sse "/live"

# remove previous policies
check "Delete policy for 127.0.0.1"
delete_policy_if_exists "127.0.0.1"

check "Policies endpoint returns an empty array"
get_data "/policy" "[]"

check "Create a policy returns 201"
post_json '{"ip":"127.0.0.1","action":"ignore"}' "/policy" 201

check "Policies endpoint returns a single record"
get_data "/policy" '[{"ip":"127.0.0.1","action":"ignore"}]'

# --- summary ---

printf "\n%d passed, %d failed\n" "$PASS" "$FAIL"
[ "$FAIL" -eq 0 ]