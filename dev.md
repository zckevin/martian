## load modifiers group
curl -x localhost:9000 -X POST -H "Content-Type: application/json" -d @modifier.json "http://martian.proxy/configure"

## Test 302 redirect
curl "https://httpbin--org.local.host/org/httpbin/ST/m30l7zQ8bj3QmedCUrV7bXKw/////////redirect-to?url=https%3A%2F%2Fwww.zhihu.com%2Fsignin%3Fnext%3D%252F" \
    -x http://localhost:9000 \
    --proxy-insecure --insecure -v

## Test set-cookie
curl "https://baidu--com.local.host/com/baidu/xueshu/ST/m30l7zQ8bj3QmedCUrV7bXKw/////////" \
    -x http://localhost:9000 \
    --proxy-insecure --insecure -v