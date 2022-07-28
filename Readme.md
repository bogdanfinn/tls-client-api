# TLS-Client-API

### Preface

This is an application which is using [gosoline](https://github.com/justtrackio/gosoline) and [TLS-Client](https://github.com/bogdanfinn/tls-client) to run a simple request forwarding service with the option to use specific tls fingerprints which are implemented in [TLS-client](https://github.com/bogdanfinn/tls-client).

### Supported Clients

- chrome_103
- safari_15_3
- safari_15_5
- safari_ios_15_5
- firefox_102
- opera_89

#### Need other clients?

Please open an issue on [this](https://github.com/bogdanfinn/tls-client) github repository. In the best case you provide the response of https://tls.peet.ws/api/all requested by the client you want to be implemented.

### Build API
Make sure to also checkout this repository `https://github.com/Solem8s/gosoline` on the branch `tls-client-api` next to this project.
Afterwards you can just run the following script: `cmd/tls-client-api/build.sh` or use the prebuilt binaries in `dist`

### Configuration & Start
* Configure stuff like api port and authentication keys in the `cmd/tls-client-api/config.dist.yml` file.
* The endpoint is `http://127.0.0.1:8080/api/forward`
* You need to set a `x-api-key` header with an auth key from the config file.

### How to use this api when it is running
You need to do a POST Request against this running API Service with the following JSON Request Body:
```json
{
    "tlsClientIdentifier": "chrome_103", // default chrome_103 when key omitted
    "proxyUrl": "", // use null for no proxy or omitt this key
    "headerOrder": [
        "key1",
        "key2"
    ],
    "headers": {
        "key1": "value1",
        "key2": "value2"
    },
    "requestCookies": {
        "key1": "value1",
        "key2": "value2"
    },
    "requestUrl": "https://tls.peet.ws/api/all",
    "requestBody": "", // use null for no request body or omitt this key
    "requestMethod": "GET"
}
```

#### JavaScript Fetch
```js
var myHeaders = new Headers();
myHeaders.append("x-api-key", "my-auth-key-1");
myHeaders.append("Content-Type", "application/json");

var raw = JSON.stringify({
  "tlsClientIdentifier": "chrome_103",
  "requestUrl": "https://tls.peet.ws/api/all",
  "requestMethod": "GET"
});

var requestOptions = {
  method: 'POST',
  headers: myHeaders,
  body: raw,
  redirect: 'follow'
};

fetch("127.0.0.1:8080/api/forward", requestOptions)
  .then(response => response.text())
  .then(result => console.log(result))
  .catch(error => console.log('error', error));
```

#### Python Requests
```python
import requests
import json

url = "127.0.0.1:8080/api/forward"

payload = json.dumps({
  "tlsClientIdentifier": "chrome_103",
  "requestUrl": "https://tls.peet.ws/api/all",
  "requestMethod": "GET"
})
headers = {
  'x-api-key': 'my-auth-key-1',
  'Content-Type': 'application/json'
}

response = requests.request("POST", url, headers=headers, data=payload)

print(response.text)
```
#### CURL
```curl
curl --location --request POST '127.0.0.1:8080/api/forward' \
--header 'x-api-key: my-auth-key-1' \
--header 'Content-Type: application/json' \
--data-raw '{
    "tlsClientIdentifier": "chrome_103",
    "requestUrl": "https://tls.peet.ws/api/all",
    "requestMethod": "GET"
}'
```

### Questions?

Contact me on discord