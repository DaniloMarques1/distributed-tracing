# What we need to build

We want to have an html with a form where user inputs numbers and
select the operator they want to apply. When the user submits, we will call
an api service that would return to us the response for this operation.

We need an api service that routes the request to specific api to do the math.

Client -> API Service -> Back end math operators

We need to create four apis (servers) that will be able to receive requests on:

- add
- multiply
- subtract
- divide

Client would request localhost:8080, but since the operators are four different services,
the api service is the one that knows which one to call.
