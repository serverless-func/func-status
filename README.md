func-status
----
> Inspired by [Statsig's Open-Source Status Page](https://github.com/statsig-io/statuspage)
> and [Gatus](https://github.com/TwiN/gatus)
----
## 监控项配置

```yaml
endpoints:
  - name: website                 # Name of your endpoint, can be anything
    url: "https://twin.sh/health"
    conditions:
      - "[STATUS] == 200"         # Status must be 200
      - "[BODY].status == UP"     # The json path "$.status" must be equal to UP
      - "[RESPONSE_TIME] < 300"   # Response time must be under 300ms

  - name: make-sure-header-is-rendered
    url: "https://example.org/"
    conditions:
      - "[STATUS] == 200"                          # Status must be 200
      - "[BODY] == pat(*<h1>Example Domain</h1>*)" # Body must contain the specified header
```

### Conditions

Here are some examples of conditions you can use:

| Condition                        | Description                                         | Passing values             | Failing values    |
|:---------------------------------|:----------------------------------------------------|:---------------------------|-------------------|
| `[STATUS] == 200`                | Status must be equal to 200                         | 200                        | 201, 404, ...     |
| `[STATUS] < 300`                 | Status must lower than 300                          | 200, 201, 299              | 301, 302, ...     |
| `[STATUS] <= 299`                | Status must be less than or equal to 299            | 200, 201, 299              | 301, 302, ...     |
| `[STATUS] > 400`                 | Status must be greater than 400                     | 401, 402, 403, 404         | 400, 200, ...     |
| `[STATUS] == any(200, 429)`      | Status must be either 200 or 429                    | 200, 429                   | 201, 400, ...     |
| `[CONNECTED] == true`            | Connection to host must've been successful          | true                       | false             |
| `[RESPONSE_TIME] < 500`          | Response time must be below 500ms                   | 100ms, 200ms, 300ms        | 500ms, 501ms      |
| `[IP] == 127.0.0.1`              | Target IP must be 127.0.0.1                         | 127.0.0.1                  | 0.0.0.0           |
| `[BODY] == 1`                    | The body must be equal to 1                         | 1                          | `{}`, `2`, ...    |
| `[BODY].user.name == john`       | JSONPath value of `$.user.name` is equal to `john`  | `{"user":{"name":"john"}}` |                   |
| `[BODY].data[0].id == 1`         | JSONPath value of `$.data[0].id` is equal to 1      | `{"data":[{"id":1}]}`      |                   |
| `[BODY].age == [BODY].id`        | JSONPath value of `$.age` is equal JSONPath `$.id`  | `{"age":1,"id":1}`         |                   |
| `len([BODY].data) < 5`           | Array at JSONPath `$.data` has less than 5 elements | `{"data":[{"id":1}]}`      |                   |
| `len([BODY].name) == 8`          | String at JSONPath `$.name` has a length of 8       | `{"name":"john.doe"}`      | `{"name":"bob"}`  |
| `has([BODY].errors) == false`    | JSONPath `$.errors` does not exist                  | `{"name":"john.doe"}`      | `{"errors":[]}`   |
| `has([BODY].users) == true`      | JSONPath `$.users` exists                           | `{"users":[]}`             | `{}`              |
| `[BODY].name == pat(john*)`      | String at JSONPath `$.name` matches pattern `john*` | `{"name":"john.doe"}`      | `{"name":"bob"}`  |
| `[BODY].id == any(1, 2)`         | Value at JSONPath `$.id` is equal to `1` or `2`     | 1, 2                       | 3, 4, 5           |
| `[CERTIFICATE_EXPIRATION] > 48h` | Certificate expiration is more than 48h away        | 49h, 50h, 123h             | 1h, 24h, ...      |
| `[VERSION] ~1.2.3`               | Tilde Range Comparisons (Patch)                     | >= 1.2.3, < 1.3.0          | 1.2.0, 1.3.0, ... |
| `[VERSION] ^1.2.3`               | Caret Range Comparisons (Major)                     | >= 1.2.3, < 2.0.0          | 1.2.0, 2.0.1, ... |

#### Placeholders

| Placeholder                | Description                                                                               | Example of resolved value                    |
|:---------------------------|:------------------------------------------------------------------------------------------|:---------------------------------------------|
| `[STATUS]`                 | Resolves into the HTTP status of the request                                              | `404`                                        |
| `[RESPONSE_TIME]`          | Resolves into the response time the request took, in ms                                   | `10`                                         |
| `[IP]`                     | Resolves into the IP of the target host                                                   | `192.168.0.232`                              |
| `[BODY]`                   | Resolves into the response body. Supports JSONPath.                                       | `{"name":"john.doe"}`                        |
| `[CONNECTED]`              | Resolves into whether a connection could be established                                   | `true`                                       |
| `[CERTIFICATE_EXPIRATION]` | Resolves into the duration before certificate expiration (valid units are "s", "m", "h".) | `24h`, `48h`, 0 (if not protocol with certs) |
| `[DNS_RCODE]`              | Resolves into the DNS status of the response                                              | `NOERROR`                                    |
| `[VERSION]`                | Resolves into the Version Check of the response                                           | `1.2.3`                                      |

#### Functions

| Function | Description                                                                                                                                                                                                                         | Example                            |
|:---------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|:-----------------------------------|
| `len`    | If the given path leads to an array, returns its length. Otherwise, the JSON at the given path is minified and converted to a string, and the resulting number of characters is returned. Works only with the `[BODY]` placeholder. | `len([BODY].username) > 8`         |
| `has`    | Returns `true` or `false` based on whether a given path is valid. Works only with the `[BODY]` placeholder.                                                                                                                         | `has([BODY].errors) == false`      |
| `pat`    | Specifies that the string passed as parameter should be evaluated as a pattern. Works only with `==` and `!=`.                                                                                                                      | `[IP] == pat(192.168.*)`           |
| `any`    | Specifies that any one of the values passed as parameters is a valid value. Works only with `==` and `!=`.                                                                                                                          | `[BODY].ip == any(127.0.0.1, ::1)` |

> 💡 Use `pat` only when you need to. `[STATUS] == pat(2*)` is a lot more expensive than `[STATUS] < 300`.