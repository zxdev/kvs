# server

Basic bare bones net/http KVS cache server.

    root / return 400 bad request as does any unrecognized request

    heartbeat /hb returns a 200 reponse

* The server uses a mutex so it is safe to read/write concurrently.
* The server does not implement https at this time.
* The dockerfile will start up as localhost:80 and with and HOST= environment variable configurd the server will start listening on localhost:1455.

## Memory Considerations

The KVS is a highly compresses and performant in memory database capabable of millions of reads per second with concurrent writes at around 1 million operations per second and modifies the internal structure to accomidate insertions. 

    KEON memory requirement is 8 bytes per item + container size performance offset of 2.5%
    
    1m == 1,000,000 + 25,000 entries x 8 bytes = 7.8Mb

    KEVA memory requirement is 16 bytes per item + container size performacne offset of 2.5% 
    
    1m == 1,000,000 + 25,000 entries x 16 bytes = 15.4Mb

    Note: it is possible to configure the database compaction level to 99.7% with little performance impact, however the current implementation uses the out of the box default configuration. Therefore, with larger size databases it may be advisable (as a todo) to modity the defaults or to add default configuration automatic changes based on database size and performance characteristics. 
    
## Endpoints



### Common Endpoints
* **/create/{size}**

        '1m' (one million)
        '10m' (ten million) 
        '100m' (100 million)
        custom 'n' size 
    
        'reset' restores most recent size configuration

        'drop' or '0' will clear database from memory
    
* **/stats** 

    ```golang
        type Stats struct {
            Count   int `json:"count,omitempty"`
            Max     int `json:"max,omitempty"`
            Percent int `json:"percent,omitempty"`
        }
    ```


### KEON specific endpoints

    200 success
    503 unavailable when no database configured

    GET single item
    POST multiple items requires an \n delimited list of items in request body

        item_1
        item_2
        ...
        item_n

* **GET /insert/{key}**
* **GET /remove/{key}** 
* **GET /check/{key}**

    json response format
    ```golang
        type Job struct {
            Status bool   `json:"status,omitempty"`
            Key    string `json:"key,omitempty"`
        }
    ```

bulk item insertion requires an \n delimited list of items as the post body payload

* **POST /insert** 
* **POST /remove** 
* **POST /check** 

    json response format
    ```golang
        type Job []struct {
            Status bool   `json:"status,omitempty"`
            Key    string `json:"key,omitempty"`
        }
    ```

### KEVA specific endpoints

    {string:uint64} is the key:value block format
    the value can represent anything and using a simple codex information can be stored as a numerically equivalent bit object|flag set

    200 success
    503 unavailable when no database configured

    GET single item {string:uint64}

    POST multiple items requires an \n delimited list of items in request body

        {key_1:123}
        {key_2:456}
        ...
        {key_n:13579}


* **GET /insert/{key:uint64}**
* **GET /remove/{key}** 
* **GET /check/{key}**

    json response format
    ```golang
        type Job struct {
            Status bool   `json:"status,omitempty"`
            Key    string `json:"key,omitempty"`
            Value  uint64 `json:"value,omitempty"`
        }
    ```

bulk POST verbs requires an \n delimited list of items as the post body payload

        {key_1_1:123}
        {key_2:456}
        ...
        {key_n:13579}

* **POST /insert** 
* **POST /remove** 
* **POST /check** 

    json repsonse format
    ```golang
        type Job []struct {
            Status bool   `json:"status,omitempty"`
            Key    string `json:"key,omitempty"`
            Value  uint64 `json:"value,omitempty"`
        }
    ```
