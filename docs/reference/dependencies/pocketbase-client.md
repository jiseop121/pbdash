# pocketbase-client Guide

이 문서는 `github.com/mrchypark/pocketbase-client`를 이 프로젝트에서 어떻게 사용하는지 정리한 상세 문서다.

## 목적
- PocketBase 연동 변경 시 가장 먼저 확인해야 하는 SDK 사용 지점을 모은다.
- 인증, 조회, 오류 매핑이 어디서 결정되는지 빠르게 찾을 수 있게 한다.

## 사용 위치
- `internal/pocketbase/client.go`
- `internal/pocketbase/query.go`

## 이 프로젝트에서 사용하는 핵심 API
- `pbclient.NewClient(...)`
- `pbclient.WithHTTPClient(...)`
- `Client.WithToken(...)`
- `Client.Send(...)`
- `*pbclient.Error`
- `pbclient.IsAuthenticationFailed(...)`

## 현재 사용 방식
- `NewClient()`에서 기본 `http.Client` 타임아웃을 `15s`로 둔다.
- 인증은 `_superusers/auth-with-password`를 먼저 시도하고, 필요 시 legacy `admins/auth-with-password`로 fallback 한다.
- 조회 요청은 GET + JSON 응답 파싱 중심으로만 사용한다.
- SDK 에러는 `AuthError` 또는 `APIError`로 다시 매핑해서 앱 전반의 오류 표현을 고정한다.
- 토큰은 `Bearer ` prefix를 보정한 뒤 Authorization 헤더에 넣는다.

## 코드 흐름
1. `Authenticate()`가 SDK 클라이언트를 만든다.
2. `_superusers` 인증 엔드포인트를 먼저 호출한다.
3. 실패가 인증 오류면 즉시 반환하고, 그 외에는 legacy admin 엔드포인트를 이어서 시도한다.
4. 응답에서 `token`을 꺼내고 비어 있으면 실패 처리한다.
5. `GetJSON()`은 endpoint와 query를 조립한 뒤 GET 요청을 보낸다.
6. 응답 본문은 object 또는 list로 파싱하고, list면 `items` 키 아래로 정규화한다.

## 업데이트 시 주의사항
- 인증 에러 판정 방식이 바뀌면 `mapSDKError()`가 먼저 깨질 수 있다.
- `Send()`의 응답 파싱 방식이 바뀌면 `Authenticate()`와 `GetJSON()` 동작을 같이 점검해야 한다.
- `_superusers`와 `admins` 경로 호환성은 PocketBase 서버 버전 변화에 민감할 수 있다.
- 이 프로젝트는 현재 PocketBase 쓰기 흐름보다 읽기 중심이라, SDK를 확장 도입할 때도 기본 방향은 read-only에 가깝게 유지한다.

## 간접 의존성 해석
- `github.com/pocketbase/pocketbase`는 현재 직접 import 하지 않는다.
- 실제 통합 지점은 이 SDK이며, PocketBase 서버 자체는 런타임 또는 E2E 대상이다.
- PocketBase 관련 문제를 볼 때는 먼저 SDK 사용 코드(`internal/pocketbase`)를 보고, 그 다음 서버 버전 호환성을 본다.

## 변경 후 체크리스트
1. `go test ./...`를 실행한다.
2. 인증 성공/실패 흐름을 확인한다.
3. collections 조회와 records 조회를 확인한다.
4. 네트워크 오류와 인증 오류가 기존과 같은 오류 타입으로 매핑되는지 확인한다.
5. 사용자에게 노출되는 에러 메시지 계약이 바뀌지 않았는지 확인한다.
