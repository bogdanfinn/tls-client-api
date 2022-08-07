# TLS-Client-API

### Preface

This is an application which is using [gosoline](https://github.com/justtrackio/gosoline) and [TLS-Client](https://github.com/bogdanfinn/tls-client) to run a simple request forwarding service with the option to use specific tls fingerprints which are implemented in [TLS-client](https://github.com/bogdanfinn/tls-client).

### Supported Clients

- chrome_104
- chrome_103
- safari_15_3
- safari_15_5
- safari_ios_15_5
- firefox_102
- opera_89

#### Need other clients?

Please open an issue on [this](https://github.com/bogdanfinn/tls-client) github repository. In the best case you provide the response of https://tls.peet.ws/api/all requested by the client you want to be implemented.

### Use API
You can just run the prebuilt binaries in `dist`. There is a binary for linux, macos and windows. Just modify your config file next to the binary as explained below and start the application.

### Build API from source
When you want to build the application from source, make sure to also checkout this repository `https://github.com/Solem8s/gosoline` on the branch `tls-client-api` next to this project.
Afterwards you can just run the following script: `cmd/tls-client-api/build.sh SOME_BUILD_IDENTIFIER` and it should build the binaries for you

### Configuration & Start
* Configure stuff like api port and authentication keys in the `cmd/tls-client-api/config.dist.yml` file.
* The endpoint is `http://127.0.0.1:8080/api/forward`
* You need to set a `x-api-key` header with an auth key from the config file. This is for protecting the API when you host it on some server. Requests without the correct keys in the header will be rejected.

### Attention
* Applications powered with [gosoline](https://github.com/justtrackio/gosoline) automatically host a health check endpoint which is by default on port `8090` under the path `/health`. So in our case it would be `http://127.0.0.1:8090/health`.
* Applications powered with [gosoline](https://github.com/justtrackio/gosoline) automatically host a metadata server for your application to provide insights into your application. The metadata server is hosted on port `8070` and has three endpoints. `/`, `/config`, `/memory` this should help you debugging your application. 
**Do not make this endpoints public available when you host the Application on some server in the internet. You would make your config file public available.**

### How to use this api when it is running
You need to do a POST Request against this running API Service with the following JSON Request Body:
```json
{
  "tlsClientIdentifier": "chrome_104",
  "proxyUrl": "",
  "headerOrder": [
    "key1",
    "key2"
  ],
  "headers": {
    "key1": "value1",
    "key2": "value2"
  },
  "requestCookies": [
    {
      "name": "cookieName",
      "value": "cookieValue",
      "path": "cookiePath",
      "domain": "cookieDomain",
      "expires": "cookieExpires"
    }
  ],
  "requestUrl": "https://tls.peet.ws/api/all",
  "requestBody": "",
  "requestMethod": "GET"
}
```
* If `tlsClientIdentifier` is not specified chrome_104 will be used
* If you do not want to set `requestBody` or `proxyUrl` use `null` instead of empty string
* Header order might be random when no order is specified

#### JavaScript Fetch
```js
var myHeaders = new Headers();
myHeaders.append("x-api-key", "my-auth-key-1");
myHeaders.append("Content-Type", "application/json");

var raw = JSON.stringify({
  "tlsClientIdentifier": "chrome_104",
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
  "tlsClientIdentifier": "chrome_104",
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
    "tlsClientIdentifier": "chrome_104",
    "requestUrl": "https://tls.peet.ws/api/all",
    "requestMethod": "GET"
}'
```

### Frequently Asked Questions / Errors
Please refer to https://github.com/bogdanfinn/tls-client#frequently-asked-questions--errors

### Questions?

Contact me on discord