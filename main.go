package main

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/alecthomas/jsonschema"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/textileio/go-threads/api/client"
	"github.com/textileio/go-threads/core/thread"
	"github.com/textileio/go-threads/db"
	"github.com/textileio/textile/api/common"
	"google.golang.org/grpc"
)

type Person struct {
	ID        string `json:"_id"`
	Name      string `json:"name"`
	Age       int    `json:"age"`
	CreatedAt int64  `json:"created_at"`
}

func GetRandomUser() (thread.Identity, error) {
	privateKey, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		return nil, err
	}
	myIdentity := thread.NewLibp2pIdentity(privateKey)
	return myIdentity, nil
}

func NewUserAuthCtx(ctx context.Context, userGroupKey string, userGroupSecret string) (context.Context, error) {
	// Add our user group key to the context
	ctx = common.NewAPIKeyContext(ctx, userGroupKey)

	// Add a signature using our user group secret
	return common.CreateAPISigContext(ctx, time.Now().Add(time.Minute), userGroupSecret)
}

func NewTokenCtx(ctx context.Context, cli *client.Client, user thread.Identity) (context.Context, error) {
	// Generate a new token for the user
	token, err := cli.GetToken(ctx, user)
	if err != nil {
		return nil, err
	}
	return thread.NewTokenContext(ctx, token), nil
}

func main() {
	// creds := credentials.NewTLS(&tls.Config{})
	// auth := common.Credentials{}
	// opts := []grpc.DialOption{grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(auth)}

	cli, err := client.NewClient("127.0.0.1:6006", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	user, err := GetRandomUser()
	if err != nil {
		panic(err)
	}

	//For using this, we must create an account on the hub. https://docs.textile.io/hub/apis/
	// authCtx, err := NewUserAuthCtx(context.Background(), "<key>", "<secret>")
	// if err != nil {
	// 	panic(err)
	// }

	tokenCtx, err := NewTokenCtx(context.Background(), cli, user)
	if err != nil {
		panic(err)
	}

	// Generate a new thread ID
	threadID := thread.NewIDV1(thread.Raw, 32)

	// Create your new thread
	err = cli.NewDB(tokenCtx, threadID)
	if err != nil {
		panic(err)
	}

	fmt.Println("> Success!")
	fmt.Println(threadID)

	reflector := jsonschema.Reflector{}
	mySchema := reflector.Reflect(&Person{}) // Generate a JSON Schema from a struct

	err = cli.NewCollection(context.Background(), threadID, db.CollectionConfig{
		Name:   "Persons",
		Schema: mySchema,
		Indexes: []db.Index{{
			Path:   "name", // Value matches json tags
			Unique: true,   // Create a unique index on "name"
		}},
	})
	if err != nil {
		panic(err)
	}

	brian := &Person{
		ID:        "",
		Name:      "Brian",
		Age:       30,
		CreatedAt: time.Now().UnixNano(),
	}

	personId := createPerson(brian, cli, threadID)

	fmt.Println(personId)

	person := &Person{}
	err = cli.FindByID(context.Background(), threadID, "Persons", personId, person)
	if err != nil {
		panic(err)
	}

	fmt.Println(person.Name)
}

func createPerson(person *Person, cli *client.Client, threadID thread.ID) string {
	ids, err := cli.Create(context.Background(), threadID, "Persons", client.Instances{person})
	if err != nil {
		panic(err)
	}
	return ids[0]
}
