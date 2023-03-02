# TLS-Client-API

### Preface

This is an application which is using [gosoline](https://github.com/justtrackio/gosoline) and [TLS-Client](https://github.com/bogdanfinn/tls-client) to run a simple request forwarding service with the option to use specific tls fingerprints which are implemented in [TLS-client](https://github.com/bogdanfinn/tls-client).

### Supported Clients
- chrome_110
- chrome_109
- chrome_108
- chrome_107
- chrome_106
- chrome_105
- chrome_104
- chrome_103
- safari_15_6_1
- safari_16_0
- safari_ipad_15_6
- safari_ios_15_5
- safari_ios_15_6
- safari_ios_16_0
- firefox_102
- firefox_104
- firefox_105
- firefox_106
- firefox_108
- firefox_110
- opera_89
- opera_90
- opera_91
- zalando_android_mobile
- zalando_ios_mobile
- nike_ios_mobile
- nike_android_mobile
- cloudscraper
- mms_ios
- mesh_ios_1
- mesh_ios_2
- mesh_android_1
- mesh_android_2

See: https://github.com/bogdanfinn/tls-client#supported-and-tested-clients

All Clients support Random TLS Extension Order by setting the option on the Http Client itself `"withRandomTLSExtensionOrder": true`.
This is needed for Chrome 107+

#### Need other clients?

Please open an issue on [this](https://github.com/bogdanfinn/tls-client) github repository. In the best case you provide the response of https://tls.peet.ws/api/all requested by the client you want to be implemented.

### Use API
You can just run the prebuilt binaries in the `dist` directory. There is a binary for linux, macos and windows. Just modify your `config.dist.yml` file next to the binary as explained below and start the application.

### Build API from source
When you want to build the application from source, make sure to also checkout this repository `https://github.com/Solem8s/gosoline` on the branch `tls-client-api` next to this project.
Afterwards you can just run the following script: `cmd/tls-client-api/build.sh SOME_BUILD_IDENTIFIER` and it should build the binaries for you.

### Configuration & Start
* Configure things like api port and authentication keys in the `cmd/tls-client-api/config.dist.yml` file.
* The default endpoint is `http://127.0.0.1:8080/api/forward`
* You need to set a `x-api-key` header with an auth key from the config file. This is for protecting the API when you host it on some server. Requests without the correct keys in the header will be rejected.

### Attention
* Applications powered with [gosoline](https://github.com/justtrackio/gosoline) automatically host a health check endpoint which is by default on port `8090` under the path `/health`. So in our case it would be `http://127.0.0.1:8090/health`.
* Applications powered with [gosoline](https://github.com/justtrackio/gosoline) automatically host a metadata server for your application to provide insights into your application. The metadata server is hosted on port `8070` and has three endpoints. `/`, `/config`, `/memory` this should help you debugging your application. 
**Do not make this endpoints public available when you host the Application on some server in the internet. You would make your config file public available.**

### How to use this api when it is running
You need to do a POST Request against this running API with parts of the following JSON Request Body (keep in mind this is the full body - not the minimum required):
```json
{
  "sessionId": "reusableSessionId",
  "tlsClientIdentifier": "chrome_105",
  "followRedirects": false,
  "insecureSkipVerify": false,
  "withRandomTLSExtensionOrder": false,
  "isByteRequest": false,
  "withoutCookieJar": false,
  "catchPanics": false,
  "additionalDecode": null,
  "withDefaultCookieJar": false,
  "withDebug": false,
  "forceHttp1": false,
  "isByteResponse": false,
  "timeoutSeconds": 30,
  "timeoutMilliseconds": 0,
  "customTlsClient": {
    "ja3String": "771,2570-4865-4866-4867-49195-49199-49196-49200-52393-52392-49171-49172-156-157-47-53,2570-0-23-65281-10-11-35-16-5-13-18-51-45-43-27-17513-2570-21,2570-29-23-24,0",
    "h2Settings": {
      "HEADER_TABLE_SIZE": 65536,
      "MAX_CONCURRENT_STREAMS": 1000,
      "INITIAL_WINDOW_SIZE": 6291456,
      "MAX_HEADER_LIST_SIZE": 262144
    },
    "h2SettingsOrder": [
      "HEADER_TABLE_SIZE",
      "MAX_CONCURRENT_STREAMS",
      "INITIAL_WINDOW_SIZE",
      "MAX_HEADER_LIST_SIZE"
    ],
    "supportedSignatureAlgorithms": [
      "ECDSAWithP256AndSHA256",
      "PSSWithSHA256",
      "PKCS1WithSHA256",
      "ECDSAWithP384AndSHA384",
      "PSSWithSHA384",
      "PKCS1WithSHA384",
      "PSSWithSHA512",
      "PKCS1WithSHA512"
    ],
    "supportedVersions": ["GREASE", "1.3", "1.2"],
    "keyShareCurves": ["GREASE", "X25519"],
    "certCompressionAlgo": "brotli",
    "pseudoHeaderOrder": [
      ":method",
      ":authority",
      ":scheme",
      ":path"
    ],
    "connectionFlow": 15663105,
    "priorityFrames": [{
      "streamID": 1,
      "priorityParam": {
        "streamDep": 1,
        "exclusive": true,
        "weight": 1
      }
    }],
    "headerPriority": {
      "streamDep": 1,
      "exclusive": true,
      "weight": 1
    }
  },
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
  "requestBody": "", // needs to be a string!
  "requestMethod": "GET"
}
```

* If `tlsClientIdentifier` is not specified chrome_108 will be used.
* You can use your own client by providing `customTlsClient` instead of `tlsClientIdentifier` 
* `sessionId` is optional. When not provided the API does not create a Session. On every forwarded request with a given sessionId you will receive the sessionId in the response to be able to reuse sessions (cookies). 
* Be aware that `insecureSkipVerify` and the `timeoutSeconds` can not be changed during a session. 
* `followRedirects` and `proxyUrl` can be changed within a session.
* If you do not want to set `requestBody` or `proxyUrl` use `null` instead of empty string
* When you set `isByteResponse` to `true` the response body will be a base64 encoded string. Useful when you want to download images for example.
* When you set `isByteRequest` to `true` the request body needs to be a base64 encoded string. Useful when you want to upload images for example.
* Header order might be random when no order is specified

#### Possible Custom client Settings
```go
var H2SettingsMap = map[string]http2.SettingID{
    "HEADER_TABLE_SIZE":      http2.SettingHeaderTableSize,
    "ENABLE_PUSH":            http2.SettingEnablePush,
    "MAX_CONCURRENT_STREAMS": http2.SettingMaxConcurrentStreams,
    "INITIAL_WINDOW_SIZE":    http2.SettingInitialWindowSize,
    "MAX_FRAME_SIZE":         http2.SettingMaxFrameSize,
    "MAX_HEADER_LIST_SIZE":   http2.SettingMaxHeaderListSize,
}

var tlsVersions = map[string]uint16{
    "GREASE": tls.GREASE_PLACEHOLDER,
    "1.3":    tls.VersionTLS13,
    "1.2":    tls.VersionTLS12,
    "1.1":    tls.VersionTLS11,
    "1.0":    tls.VersionTLS10,
}

var signatureAlgorithms = map[string]tls.SignatureScheme{
    "PKCS1WithSHA256":        tls.PKCS1WithSHA256,
    "PKCS1WithSHA384":        tls.PKCS1WithSHA384,
    "PKCS1WithSHA512":        tls.PKCS1WithSHA512,
    "PSSWithSHA256":          tls.PSSWithSHA256,
    "PSSWithSHA384":          tls.PSSWithSHA384,
    "PSSWithSHA512":          tls.PSSWithSHA512,
    "ECDSAWithP256AndSHA256": tls.ECDSAWithP256AndSHA256,
    "ECDSAWithP384AndSHA384": tls.ECDSAWithP384AndSHA384,
    "ECDSAWithP521AndSHA512": tls.ECDSAWithP521AndSHA512,
    "PKCS1WithSHA1":          tls.PKCS1WithSHA1,
    "ECDSAWithSHA1":          tls.ECDSAWithSHA1,
}

var curves = map[string]tls.CurveID{
    "GREASE": tls.CurveID(tls.GREASE_PLACEHOLDER),
    "P256":   tls.CurveP256,
    "P384":   tls.CurveP384,
    "P521":   tls.CurveP521,
    "X25519": tls.X25519,
}

var certCompression = map[string]tls.CertCompressionAlgo{
    "zlib":   tls.CertCompressionZlib,
    "brotli": tls.CertCompressionBrotli,
    "zstd":   tls.CertCompressionZstd,
}
```

#### Response
The Response from the API looks like that:
```json
{
  "id": "some response identifier",
  "sessionId": "some reusable sessionId if provided on the request",
  "status": 200,
  "target": "the target url",
  "body": "The Response as string here or the error message",
  "headers": {},
  "cookies": {}
}
```
* In case of an error the status code will be 0


#### JavaScript Fetch minified example
##### Forward Request `/api/forward`
```js
var myHeaders = new Headers();
myHeaders.append("x-api-key", "my-auth-key-1");
myHeaders.append("Content-Type", "application/json");

var raw = JSON.stringify({
  "tlsClientIdentifier": "chrome_105",
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

##### Free Single Session `/api/free-session`
```js
var myHeaders = new Headers();
myHeaders.append("x-api-key", "my-auth-key-1");
myHeaders.append("Content-Type", "application/json");

var raw = JSON.stringify({
  "sessionId": "my-custom-sessionId"
});

var requestOptions = {
  method: 'POST',
  headers: myHeaders,
  body: raw,
  redirect: 'follow'
};

fetch("127.0.0.1:8080/api/free-session", requestOptions)
  .then(response => response.text())
  .then(result => console.log(result))
  .catch(error => console.log('error', error));
```

##### Free All Sessions `/api/free-all`
```js
var myHeaders = new Headers();
myHeaders.append("x-api-key", "my-auth-key-1");

var requestOptions = {
  method: 'GET',
  headers: myHeaders,
  redirect: 'follow'
};

fetch("127.0.0.1:8080/api/free-all", requestOptions)
  .then(response => response.text())
  .then(result => console.log(result))
  .catch(error => console.log('error', error));
```

#### Python Requests minified example
##### Forward Request `/api/forward`
```python
import requests
import json

url = "127.0.0.1:8080/api/forward"

payload = json.dumps({
  "tlsClientIdentifier": "chrome_105",
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

##### Free Single Session `/api/free-session`
```python
import requests
import json

url = "127.0.0.1:8080/api/free-session"

payload = json.dumps({
  "sessionId": "my-custom-sessionId"
})
headers = {
  'x-api-key': 'my-auth-key-1',
  'Content-Type': 'application/json'
}

response = requests.request("POST", url, headers=headers, data=payload)

print(response.text)
```

##### Free All Sessions `/api/free-all`
```python
import requests

url = "127.0.0.1:8080/api/free-all"

payload={}
headers = {
  'x-api-key': 'my-auth-key-1',
}

response = requests.request("GET", url, headers=headers, data=payload)

print(response.text)
```

#### CURL minified example
##### Forward Request `/api/forward`
```curl
curl --location --request POST '127.0.0.1:8080/api/forward' \
--header 'x-api-key: my-auth-key-1' \
--header 'Content-Type: application/json' \
--data-raw '{
    "tlsClientIdentifier": "chrome_105",
    "requestUrl": "https://tls.peet.ws/api/all",
    "requestMethod": "GET"
}'
```

##### Free Single Session `/api/free-session`
```curl
curl --location --request POST '127.0.0.1:8080/api/free-session' \
--header 'x-api-key: my-auth-key-1' \
--header 'Content-Type: application/json' \
--data-raw '{
  "sessionId":"my-custom-sessionid"
}'
```

##### Free All Sessions `/api/free-all`
```curl
curl --location --request GET '127.0.0.1:8080/api/free-all' \
--header 'x-api-key: my-auth-key-1'
```

### Frequently Asked Questions / Errors
* **I can not do a successful POST Request.**

Be aware that when you do a POST Request and want to provide a forwarded request body in the `requestBody` field it has to be a string. That means if you want to send JSON you need to stringify this JSON to a string first.

* **How can I use other request body content types besides json? **

`requestBody` accepts strings and forwards them as the payload. combined with the `content-type` header the api makes the actual request body out of it. You can use for example `application/x-www-form-urlencoded` content type in the header and then just provide as request body a string similar to `key=value&key=value`

For more Questions and answers please refer to https://github.com/bogdanfinn/tls-client#frequently-asked-questions--errors

### Questions?

Join my discord support server: https: // discord.gg / 7Ej9eJvHqk
No Support in DMs!