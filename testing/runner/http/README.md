**Http runner service**

Http runner sends one or more HTTP request to the specified endpoint; 
it manages cookie within [SendRequest](service_contract.go).

| Service Id | Action | Description | Request | Response |
| --- | --- | --- | --- | --- |
| http/runner | send | Sends one or more http request to the specified endpoint. | [SendRequest](service_contract.go) | [SendResponse](service_contract.go) |


##Usage

- [Basic and conditional requests](#basic)
- [Repeating request](#repeating)
- [User defined function transformation](#udf)
- [Sequential request with data extraction](#sequential)
- [Request with validation](#assert)
- [Testing http request from cli](#cli)
- [Sending http request from inline workflow](#inline)
- [Neatly http body payload data organization](#workflow)


<a name="basic"></a>
**Basic and conditional requests execution**

```go
	import (
		"log"
		"net/http"
		"github.com/viant/endly"
		"github.com/viant/toolbox"
		runner "github.com/viant/endly/testing/runner/http"
	)
	
	
	func main() {
        response := &runner.SendResponse{}
        
        err := endly.Run(context, &runner.SendRequest{
            Options:[]*toolbox.HttpOptions{
                 {
                    Key:"RequestTimeoutMs",
                    Value:12000,
                 },	
            },
            Requests: []*runner.Request{
                {
                    URL:    "http://127.0.0.1:8111/send1",
                    Method: "POST",
                    Body:   "some body",
                    Header:http.Header{
                        "User-Agent":[]string{"myUa"},
                    },
                },
                {   //Only run the second request in previous response body contains 'content1-2' fragment
                    When:   "${httpTrips.Response[0].Body}:/content1-2/",
                    URL:    "http://127.0.0.1:8111/send2",
                    Method: "POST",
                    Body:   "other body",
                },
            },
        }, response)
        if err != nil {
            log.Fatal(err)
        }
        
    }

```

A send request represents a group of http requests, Individual request run can be optionally conditioned with **When** [criteria](../../../criteria/README.md)
Each run publishes '**httpTrips**' journal to the context.State with .Response and .Request keys representing collection about the active execution.

<a name="repeating"></a>
**Repeating request**
```go


     response := &runner.SendResponse{}
     err := endly.Run(context, &runner.SendRequest{
            Requests: []*runner.Request{
    			{
    				URL:    "http://127.0.0.1:8111/send1",
    				Method: "POST",
    				Body:   "0123456789",
    				Repeater: &model.Repeater{
    					Repeat: 10,
    					
    				},
    			}
  			}
	})


```

Simply repeat 10 times post request


@http.json
```json

{
    "Requests": [
        {
            "URL": "http://testHost/ready",
            "Method": "GET",
            "Extraction": [
                {
                    "Key": "testHostStatus",
                    "RegExpr": "Current Server Status: ([A-Z]*)"
                }
            ],
            "Repeat": 150,
            "SleepTimeMs": 2000,
            "Exit": "$testHostStatus: READY"
        }
    ]
}
```

Repeat till test host status is 'READY', keep testing for status no more than 150 times with 2 second sleep.


<a name="udf"></a>
**User defined function transformation**


```go
    request, err := runner.NewSendRequestFromURL("http.json")
	var response = &runner.SendResponse{}
	err = endly.Run(context, request, response)
	if err != nil {
		log.Fatal(err)
		return
	}
```


@http.json
```json
{
  "Requests": [
    {
      "Method": "post",
      "URL": "http://127.0.0.1:8987/xxx?access_key=abc",
      "RequestUdf": "UserAvroWriter",
      "ResponseUdf": "AvroReader",
      "JSONBody": {
        "ID":1,
        "Desc":"abc"
      }
    }
  ],
  "UdfProviders": [
    {
      "Id": "UserAvroWriter",
      "Provider": "AvroWriter",
      "Params": [
        "{\"type\": \"record\", \"name\": \"user\", \"fields\": [{\"name\": \"ID\",\"type\":\"int\"},{\"name\": \"Desc\",\"type\":\"string\"}]}"
      ]
    }
  ]
}
```


<a name="sequential"></a>
**Sequential request with data extraction**


@http.json
```json

{
	"Requests": [
		{
			"Method": "POST",
			"URL": "http://${bidderHost}/bidder",
			"Body": "$bid0",
			"Extraction": [
            				{
            					"Key": "winURI",
            					"RegExpr": "(/pixel/won[^\\']+)",
            					"Reset": true
            				},
            				{
            					"Key": "clickURI",
            					"RegExpr": "(/pixel/click[^\\']+)",
            					"Reset": true
            				}
            ],
			"Variables": [
				{
					"Name": "AUCTION_PRICE",
					"From": "seatbid[0].bid[0].price"
				},
                {
                    "Name": "winURL",
                    "Value": "http://${loggerHost}/logger${winURI}"
                },
                {
                    "Name": "clickURL",
                    "Value": "http://${loggerHost}/logger${clickURI}"
                }
			]
		},
		{
			"When": "${httpTrips.Response[0].Body}://pixel/won//",
			"URL": "${httpTrips.Data.winURL}",
			"Method": "GET"
		},
		{
			"When": "${httpTrips.Response[0].Body}://pixel/click//",
			"URL": "${httpTrips.Data.clickURL}",
			"Method": "GET"
		}
	]
}
```



The following example extracts from _POST Body_ tracking URL with regular expression matches 
and from structured POST Body _AUCTION_PRICE_ and variables that define subsequent URL. 
  
<a name="assert"></a>
**Request with validation**


@send.json
```json
{
  "Requests": [
    {
      "Method": "POST",
      "URL": "http://127.0.0.1:8080/v1/api/dummy",
      "Body": "{}"
    }
  ],
  "Expect": {
    "Responses": [
      {
        "Code": 200,
        "JSONBody": {
          "Status": "error",
          "Error": "data was empty"
        }
      }
    ]
  }
}
```


<a name="cli"></a>
**Testing http request from cli**

```bash
endly -w=action service='http/runner' action=send request='@send.json'
```


@send.json
```json
{
  "Requests": [
    {
      "Method": "POST",
      "URL": "http://127.0.0.1:8080/uri",
      "JSONBody": {
        "key1":1
      }
    }
  ]
}
```

<a name="inline"></a>
**Sending http request from inline workflow**


```bash
endly -r=inline
```


@inline.yaml
```yaml
pipeline:
  task1:
    action: http/runner:send
    request: "@send.json" 
```


<a name="workflow"></a>
**Neatly http body payload data organization**

Imagine a case where Payload body can be shared across various HTTP requests within the same group.


@http_send.json
```json
{
	"Requests": [
		{
			"Method": "POST",
			"URL": "http://${testHost}/uri",
			"Body": "$payload1"
		},
		{
            "Method": "POST",
            "URL": "http://${testHost}/uri",
            "Body": "$payload2"
        },
        {
            "Method": "POST",
            "URL": "http://${testHost}/uri",
            "Body": "$payload1"
        }
	]
}
```

@payloads.json
```json
{
  "payload1": {
    "k1":"some data here"
  },
  "payload2": {
      "k100":"some data here"
  }
}
```

You can use the following [neatly](github.com/viant/neatly) construct

@test.csv

|Workflow|Name|Tasks| | |
|---|---|---|---| --- |
| |test|%Tasks|| |
|**[]Tasks**|**Name**|**Actions**| |
| |task1|%Actions| |
|**[]Actions**|**Service**|**Action**|**Request**|**Description**|
| |http/runner|send|@http_send @payloads | send http requests |

