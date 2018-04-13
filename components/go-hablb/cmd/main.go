package main

import (
	"fmt"
	"time"

	habgrpc "github.com/jaym/hab-go-grpc/components/go-hablb/grpc"
)

func main() {
	r := habgrpc.NewResolver()
	w, err := r.Resolve("teams-service.default:port")

	if err != nil {
		panic(err)
	}

	for {
		n, err := w.Next()

		if err != nil {
			panic(err)
		}

		for _, op := range n {
			fmt.Printf("%+v\n", op)
		}
		time.Sleep(time.Second)
	}
}
