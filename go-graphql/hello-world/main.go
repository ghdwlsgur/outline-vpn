package main

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/graphql-go/graphql"
)

func main() {

	/* GraphQL이란 ?
	GraphQL은 페이스북에서 만든 쿼리 언어입니다. 국내에서 GraphQL API를 Open API로 공개한 곳은 드뭅니다. 또한, 해외의 경우,
	Github 사례(Github v4 GraphQL)를 찾을 수는 있지만, 전반적으로 GraphQL API를 Open API로 공개한 곳은 많지 않습니다.

	SQL은 데이터베이스 시스템에 저장된 데이터를 효율적으로 가져오는 것이 목적이고, GQL은 웹 클라이언트가 데이터를 서버로부터 효율적으로
	가져오는 것이 목적입니다. GQL의 문장은 주로 클라이언트 시스템에서 작성하고 호출합니다.

	GraphQL은 애플리케이션 프로그래밍 인터페이스(API)를 위한 쿼리 언어이자 서버측 런타임으로 클라이언트에게 요청한 만큼의 데이터를
	제공하는데 우선 순위를 둡니다. GraphQL은 API를 더욱 빠르고 유연하며 개발자 친화적으로 만들기 위해 설계되었습니다.

	GraphiQL이라고 하는 통합 개발 환경(IDE) 내에 배포될 수도 있습니다. REST를 대체할 수 있는 GraphQL은 개발자가 단일 API 호출로
	다양한 데이터 소스에서 데이터를 끌어오는 요청을 구성할 수 있도록 지원합니다. 또한 GraphQL은 API 유지 관리자에게 기존 쿼리에 영향을 미치지 않고
	필드를 추가하거나 폐기할 수 있는 유연성을 부여합니다. 개발자는 자신이 선호하는 방식으로 API를 빌드할 수 있으며, GraphQL 사양은 이러한 API가 예측
	가능한 방식으로 작동하도록 보장합니다.
	*/

	/* 스키마, 리졸버를 비롯한 일반적인 GraphQL 용어

	API 개발자는 GraphQL을 사용해 클라이언트가 서비스를 통해 쿼리할 가능성이 있는 모든 데이터를 설명하는 스키마를 생성합니다.
	GraphQL 스키마는 개체 유형으로 구성되어 어떤 종류의 개체를 요청할 수 있으며 어떠한 필드가 있는지 정의합니다. 쿼리가 수신되면
	GraphQL은 스키마에 대해 쿼리를 검증하고 그 다음 검증된 쿼리를 실행합니다.

	API 개발자는 스키마의 각 필드를 리졸버라고 불리는 기능에 첨부합니다. 실행 중 값을 생산하기 위해 리졸버가 호출됩니다.
	*/

	// 객체 필드는 해당 객체에 대한 데이터나 다른 객체와의 연결에 대한 데이터를 보여준다.
	fields := graphql.Fields{
		"hello": &graphql.Field{
			Type: graphql.String,
			Resolve: func(p graphql.ResolveParams) (interface{}, error) {
				return "world", nil
			},
		},
	}

	rootQuery := graphql.ObjectConfig{
		Name:   "RootQuery",
		Fields: fields,
	}

	schemaConfig := graphql.SchemaConfig{
		Query: graphql.NewObject(rootQuery),
	}

	schema, err := graphql.NewSchema(schemaConfig)
	if err != nil {
		log.Fatalf("failed to create new schema, error: %v", err)
	}

	query := `
					{
								hello
					}
	`

	params := graphql.Params{Schema: schema, RequestString: query}
	r := graphql.Do(params)
	if len(r.Errors) > 0 {
		log.Fatalf("failed to execute graphql operation, errors: %+v", r.Errors)
	}
	rJSON, _ := json.Marshal(r)
	fmt.Printf("%s \n", rJSON)
}
