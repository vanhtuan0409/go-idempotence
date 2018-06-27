# Go Idempotence API

Thread-safe example for [Stripe Idempotency API](https://stripe.com/blog/idempotency)

### How to start

- Install Go dependencies

```
go get ./...
```

or

```
dep ensure
```

- Bootstrap infrastructure (using docker)

```
docker-compose up -d
```

- Start server

```
go run main.go
```

### Description

Simple bidding API. Next bid must be bigger than current bid

| Route | Method | Body                                                                  |
| ----- | ------ | --------------------------------------------------------------------- |
| /bid  | GET    | None                                                                  |
| /bid  | POST   | <ul><li>Idempotence-Key (Header)</li><li>Value (json:value)</li></ul> |
