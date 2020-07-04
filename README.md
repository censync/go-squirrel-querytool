The root structure for select request

```
{
   "filters":{},
   "sorting":[],
   "limit":100,
   "offset":0
}
```
Every field may be ommited for using default params.

### Available filtering operators ###

***Strict selection***

Examples for using strict operator `=` (equal):   
`{"username": "john_doe"}` is similar `"{username": { "=": "john_doe" }}`  
For non string types: `{"status": 2}`, `{"is_moderated": false}`  

Examples for using strict operator`!=` (not equal):  
`{"username": { "!=": "john_smith" }}`, `{"status": { "!=": 3 }}`, `{"is_moderated": { "!=": true }}`  


***List entry selection***

Examples for using operator `in`:  
`{"country": { "in": [ "ru", "us", "es" ]}}`, `{"status": { "in": [3, 4, 5] }}`  
Examples for using operator `not in`:  
`{"country": { "not in": [ "ua", "kz" ] }}`, `{"sex": { "in": [0, 1]}}`  


***Conditional selection***

Available operators: 

* "lt" less than "<"
* "gt" greater than ">"
* "lte" greater than "<="
* "gte" greater than ">="

`{"balance": { "lte": 4.3242 }}`, `{"size": { "lt": 15 }}`   


***Wildcard string selection***

`{"username": { "like": "%super%" }}`, `{"firstname": { "like": "John S%" }}`

### Using sorting ###

Sorting may get string array `column direction` or array of key-value`{"column" : "direction"}`, but do not mix it:
Examples: `["id DESC", "balance ASC"]` or `[{"id" : "DESC"}, {"balance" : "ASC"}]`

### Example JSON request ###

```
{
   "filters":{
      "phone":{
         "like":"+1213%"
      },
      "sex":1,
      "firstname":{
         "in":[
            "John",
            "Jake"
         ]
      }
   },
   "sorting":[
      {"id": "DESC"}
   ],
   "limit":200,
   "offset":0
}
```

### Example QUERY request ###

```
http://example.com/list?filters[phone][like]=+1213%&filters[sex]=1&filters[firstname][in][]=John&filters[firstname][in][]=Jake&sorting[0][id]=DESC&limit=200&offset=0
```
### Example code ###
```go
package requests

import (
	querytool "github.com/censync/go-squirrel-querytool"
)

type UsersListQueryForm struct {
	querytool.Query
}

```
### Binding JSON request ###

```go
func GetUsersList(c echo.Context) error {
   request := &req.UsersListQueryForm{}

   err := request.BindQuery(c.QueryParams())

   if err != nil {
	return err
   }
   
   users := service.UserService().GetUsersList(request)
   return c.JSON(200, users)
}
```
### Binding query request ###
```
```go
func GetUsersList(c echo.Context) error {
   request := &req.UsersListQueryForm{}

   err := c.Bind(request)

   if err != nil {
	return err
   }
   
   users := service.UserService().GetUsersList(request)
   return c.JSON(200, users)
}
```