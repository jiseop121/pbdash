# Multi PocketBase UI - 구현 명세서 (MVP)

이 문서는 **주니어 개발자가 그대로 구현할 수 있도록** 작성한 실행 명세다.
정책 배경보다 구현 기준을 우선한다.

관련 문서:
- 제품 요약: `README.md`
- 타입/상태 계약: `docs/spec-contracts.md`
- UI 디자인 상세: `docs/ui-design-spec.md`

---

## 0) 문서 사용법

- 구현 중 판단이 필요하면 이 문서의 "결정 규칙"을 우선 적용한다.
- 타입 정의는 `docs/spec-contracts.md`를 단일 소스로 사용한다.
- 이 문서와 타입 문서가 충돌하면, 타입 문서를 수정하지 말고 이 문서를 먼저 업데이트한다.

---

## 1) MVP 범위

### 1.1 포함(반드시 구현)
- 인스턴스 등록/수정/삭제
- 관리자 로그인(인증 메모리 세션)
- 컬렉션 목록 조회/검색/즐겨찾기
- 테이블 목록 조회, 필터, 정렬, 페이징
- 컬럼 표시/숨김, 순서 변경, Pin Left
- Inspector 상세 보기
- Table View 프리셋 저장/적용
- Workspace 저장/불러오기
- 패널 동기화(`filterQuery`, `sortSpec`)

### 1.2 제외(MVP 아님)
- 레코드 생성/수정/삭제
- 스키마 편집
- 서버 공유 저장
- 멀티 컬럼 정렬

---

## 2) 확정 결정(바꾸지 말 것)

1. 앱 형태는 Desktop이며, API 호출은 메인/백엔드 프로세스에서만 수행한다.
2. 렌더러는 IPC로만 API 요청한다.
3. 인증 정보(`token`, `adminUser`)는 메모리에만 저장하고 디스크에 저장하지 않는다.
4. 앱 재시작 시 모든 인스턴스는 `login_required` 상태에서 시작한다.
5. 401/403 발생 시 `auth_expired` 처리 후 재로그인 모달을 띄운다.
6. 재로그인 성공 시 직전 실패 요청을 1회 자동 재시도한다.
7. 동기화는 `syncGroupId` 단위로 전파하고, 실패 패널은 이전 상태를 유지한다.
8. 동기화 부분 실패 시 성공 패널은 롤백하지 않는다.

---

## 3) 화면 구조

단일 화면 App Shell 구성:

1. Top Bar
2. Left Explorer
3. Center Workspace Canvas
4. Right Inspector

정적 UI 프리뷰(`preview/*.html`) 기준:
- Right Inspector는 제거한다.
- 컬럼 구성 기능은 각 패널 내부 `Fields` 영역에서 제공한다.

### 3.1 Top Bar

필수 버튼:
- `Workspace` 드롭다운
- `Save`
- `Save As`
- `Load`
- `Instances`
- `Help`

동작 규칙:
- `Save`: 현재 활성 워크스페이스가 있으면 덮어쓰기, 없으면 Save As 모달
- `Load`: 워크스페이스 목록 모달 오픈
- `Instances`: 인스턴스 관리 모달 오픈

### 3.2 Left Explorer

필수 요소:
- 인스턴스 선택 셀렉트
- 컬렉션 검색 입력
- 즐겨찾기 섹션
- 컬렉션 목록

동작 규칙:
- 인스턴스 변경 시 해당 인스턴스 컬렉션 목록 재조회
- 컬렉션 더블클릭/Enter 시 패널 열기
- 즐겨찾기 클릭 시 해당 컬렉션 즉시 열기

### 3.3 Center Workspace Canvas

필수 요소:
- Split 트리
- 그룹별 탭 바
- 그룹별 데이터 테이블

열기 규칙:
- 첫 컬렉션 열기: Root Group 탭 추가
- 두 번째 이후: 현재 활성 그룹의 오른쪽으로 split 생성 후 새 그룹에 탭 추가

드래그 도킹 규칙:
- 드롭 존: center/left/right/top/bottom
- 그룹이 비면 split 자동 정리(부모/형제 병합)

### 3.4 Right Inspector

탭 구성:
- `Columns`
- `View`
- `Details`

동작 규칙:
- 활성 패널이 바뀌면 Inspector 컨텍스트 즉시 변경
- `Details`는 활성 패널의 선택 레코드 기준

---

## 4) 데이터 저장 규칙

### 4.1 영속 저장(LocalStorage 또는 IndexedDB)

저장 대상:
- `instances`
- `tableViewPresets`
- `workspacePresets`
- `favorites`

저장 금지:
- `token`
- `adminUser`
- 기타 인증 관련 정보

권장 키 이름:
- `mpui.instances.v1`
- `mpui.tableViewPresets.v1`
- `mpui.workspacePresets.v1`
- `mpui.favorites.v1`

### 4.2 메모리 저장(Runtime)

저장 대상:
- `authSession`
- `instanceRuntime.status`
- 임시 에러 메시지
- 요청 재시도 컨텍스트(401/403 직전 요청)

### 4.3 네이밍 규칙(필수)

아래 키 이름을 코드/저장소/문서에 동일하게 사용:
- `collectionName`
- `filterQuery`
- `sortSpec`
- `visibleColumns`
- `pinnedColumns`
- `syncGroupId`
- `appliedAt`

---

## 5) 상태 모델

타입 원본은 `docs/spec-contracts.md`를 사용한다.

필수 전역 상태:
- `activeInstanceId`
- `activePanelId`
- `activeGroupId`
- `instances: InstanceConfig[]`
- `instanceRuntime: Record<instanceId, InstanceRuntimeState>`
- `workspace: WorkspacePreset (현재 편집본)`
- `tableViewPresets: TableViewPreset[]`
- `favorites: FavoritesStore`

파생 상태:
- `activePanelState`
- `activeCollectionName`
- `isAuthReady(instanceId)`

초기화 규칙:
1. 앱 시작 시 영속 저장 데이터 로드
2. 모든 인스턴스의 `instanceRuntime.status = login_required`
3. `authSession = null`

---

## 6) 기능 상세 명세

## 6.1 인스턴스 관리

입력 필드:
- `alias`
- `baseUrl`

검증 규칙:
1. `alias`는 trim 후 공백 불가
2. `alias`는 대소문자 무시 유니크
3. `baseUrl`은 `http://` 또는 `https://`로 시작
4. `baseUrl` 끝 슬래시는 저장 시 제거

저장 동작:
- 성공 시 `instances` 영속 저장
- 현재 활성 인스턴스가 삭제되면 다음 인스턴스를 자동 선택(없으면 null)

삭제 동작:
- 삭제 시 해당 인스턴스의 `authSession` 즉시 제거
- 열린 탭 중 `instanceId`가 일치하는 탭은 즉시 닫음
- `favorites`, `tableViewPresets`에서 해당 인스턴스 데이터 제거 후 즉시 영속 저장

## 6.2 로그인/인증 만료

로그인 트리거:
- 인스턴스 상태가 `login_required` 또는 `auth_expired`
- 사용자가 로그인 버튼 클릭

성공 처리:
1. `authSession`을 메모리에 저장
2. 상태를 `ok`로 변경
3. 실패 요청 컨텍스트가 있으면 1회 자동 재시도

실패 처리:
- 상태를 유지(`login_required` 또는 `auth_expired`)
- 모달 내 에러 메시지 표시

401/403 처리:
1. 요청 실패 시 상태 `auth_expired`
2. 패널 상단 `Auth expired` 배지 표시
3. 배지 클릭 -> 로그인 모달

## 6.3 컬렉션 탐색(Explorer)

조회 조건:
- `activeInstanceId` 존재
- `instanceRuntime.status = ok`

조회 실패:
- 네트워크 실패: `network_error`
- 인증 실패: `auth_expired`

검색 규칙:
- 클라이언트 필터(컬렉션 이름 contains, 대소문자 무시)

즐겨찾기 규칙:
- 키 포맷: `${instanceId}:${collectionName}`
- 토글 시 즉시 영속 저장

## 6.4 패널/테이블

### 6.4.1 패널 열기

입력:
- `instanceId`
- `collectionName`

동작:
1. 그룹/탭 생성 규칙 적용(3.3)
2. 패널 기본 상태 생성
3. 목록 요청 실행

기본 패널 상태:
- `filterQuery = ""`
- `sortSpec = ""`
- `page = 1`
- `pageSize = 50`
- `visibleColumns` 결정 규칙:
  1. 컬렉션 필드에 `id`, `created`, `updated`가 있으면 우선 포함
  2. 위 3개 외 기본 노출 컬럼이 필요하면 컬렉션 필드 순서 기준 앞에서 최대 3개 추가
  3. 최종 컬럼 수는 최대 6개로 시작

### 6.4.2 목록 요청

요청 파라미터:
- `page`
- `pageSize`
- `filterQuery`
- `sortSpec`

성공 처리:
- 테이블 데이터 교체
- 페이지 정보 업데이트

실패 처리:
- 네트워크: 토스트 + 상태 배지
- 필터 오류: 입력창 에러 + 이전 정상 데이터 유지

### 6.4.3 필터

규칙:
- 입력 후 Enter로 적용
- `Clear`는 `filterQuery = ""` 후 재조회
- `Undo`는 마지막 정상 `filterQuery`로 복원 후 재조회

에러 규칙:
- 서버가 필터 오류를 반환하면 데이터는 바꾸지 않는다.

### 6.4.4 정렬

헤더 클릭 순환:
- `none -> asc -> desc -> none`

`sortSpec` 매핑:
- asc: `field`
- desc: `-field`
- none: `""`

### 6.4.5 페이징

버튼:
- `Prev`
- `Next`

규칙:
- `Prev`는 1페이지에서 비활성
- `Next`는 마지막 페이지에서 비활성
- 페이지 이동 시 목록 재조회

### 6.4.6 필드(컬럼) 지정

프리뷰 기준 위치:
- 각 패널의 필터/정렬/페이징 영역 아래, 테이블 위

동작 규칙:
- `Choose Columns` 버튼으로 컬럼 선택 영역을 열고 닫는다.
- 컬럼 체크박스 on/off 시 해당 테이블 컬럼을 즉시 표시/숨김 처리한다.
- 최소 1개 컬럼은 항상 표시 상태를 유지한다.
- 선택 수는 `N selected` 형태로 즉시 갱신한다.

의미 매핑:
- 이 기능은 실제 앱의 `visibleColumns` 제어와 동일한 의미를 가진다.
- MVP 본 구현에서 Inspector의 Columns 탭과 연결되는 동작을 프리뷰에서는 패널 내부로 대체한다.

## 6.5 컬럼 구성(Inspector > Columns)

필수 기능:
- 컬럼 검색
- 표시/숨김 토글
- 순서 드래그
- Pin Left 토글
- Reset to default

제약:
- 최소 1개 컬럼 항상 표시

반영 시점:
- 변경 즉시 활성 패널 테이블에 반영

## 6.6 상세 보기(Inspector > Details)

상태:
- 선택 없음: "No row selected"
- 선택 있음: 필드 목록 렌더

필수 액션:
- `Copy ID`
- `Copy Field`
- `Copy JSON`

표현 규칙:
- 객체/배열은 pretty JSON으로 렌더
- 긴 텍스트는 접기/펼치기

## 6.7 Table View 프리셋

저장 대상:
- `visibleColumns`
- `columnOrder`
- `pinnedColumns`
- `filterQuery`
- `sortSpec`
- `pageSize`

적용 범위:
- 동일 `(instanceId, collectionName)` 패널

기능:
- Save
- Apply
- Rename
- Delete

## 6.8 Workspace 프리셋

저장 대상:
- `layoutTree`
- `groups`
- 탭별 `panelState`
- 동기화 설정(`syncGroupId`)

불러오기:
1. 현재 작업 상태 확인 모달(덮어쓰기 경고)
2. 워크스페이스 로드
3. 필요한 데이터 요청 순차 실행

## 6.9 동기화(Sync)

전파 조건:
- 소스 패널 `syncGroupId` 존재
- 대상 패널 `syncGroupId` 동일

전파 필드:
- `filterQuery`
- `sortSpec`

전파 알고리즘:
1. 소스 패널 로컬 적용
2. 대상 패널들에 `Promise.allSettled`로 병렬 전파
3. 각 대상 패널 독립 성공/실패 처리

실패 처리:
- 실패 패널은 이전 정상 상태 유지
- 실패 패널에만 에러 배지 표시
- 성공 패널 롤백 금지

충돌 규칙:
- `appliedAt` 최신 이벤트가 최종 상태

---

## 7) API 어댑터 계약

실제 PocketBase SDK/HTTP 호출은 어댑터로 감싼다.

```ts
interface PocketBaseGateway {
  loginAdmin(instanceId: string, email: string, password: string): Promise<AuthSession>;
  listCollections(instanceId: string): Promise<string[]>;
  listRecords(input: {
    instanceId: string;
    collectionName: string;
    page: number;
    pageSize: number;
    filterQuery: string;
    sortSpec: string;
  }): Promise<{
    items: Record<string, unknown>[];
    page: number;
    pageSize: number;
    totalPages: number;
    totalItems: number;
  }>;
}
```

구현 규칙:
- 어댑터는 401/403/네트워크 오류를 구분 가능한 에러 타입으로 throw
- UI 레이어는 에러 타입 기반으로 상태 전이

---

## 8) 에러 UX 규칙

에러 분류:
- `NETWORK_ERROR`
- `AUTH_EXPIRED`
- `FILTER_INVALID`
- `UNKNOWN`

노출 위치:
- 전역 토스트: 네트워크/알 수 없는 오류
- 패널 인라인: 필터 오류, 동기화 대상 실패
- 패널 배지: 인증 만료

문구 규칙:
- 사용자 문구에 `baseUrl` 포함
- 기술 상세는 개발자 콘솔에만 출력

---

## 9) 구현 순서(작업 지시)

1. 저장소 계층 구현: 영속 저장/런타임 저장 분리
2. 인스턴스 CRUD + 검증
3. 로그인 모달 + 인증 상태머신
4. Explorer + 컬렉션 조회
5. Workspace/Group/Tab 기본 동작
6. 테이블 조회/필터/정렬/페이징
7. Inspector Columns/Details
8. Table View 프리셋
9. Workspace 저장/로드
10. Sync 전파/부분 실패 처리
11. 수용 테스트 수행

---

## 10) 수용 테스트(직접 실행)

### 10.1 인증 비영속
1. 인스턴스 등록 후 로그인한다.
2. 앱을 완전히 종료하고 다시 실행한다.
3. 인스턴스가 `login_required` 상태인지 확인한다.
4. 로그인 없이 데이터 요청 시 로그인 모달이 뜨는지 확인한다.

### 10.2 필터 오류 복구
1. 정상 필터를 적용해 데이터를 로드한다.
2. 의도적으로 잘못된 필터를 입력하고 Enter를 누른다.
3. 테이블 데이터가 유지되는지 확인한다.
4. `Undo` 클릭 시 직전 정상 필터로 복구되는지 확인한다.

### 10.3 동기화 부분 실패
1. 동일 `syncGroupId` 패널 2개를 준비한다.
2. 한쪽에서 유효 필터를 적용한다.
3. 다른 쪽에서 실패 조건(권한/스키마 불일치)을 만든다.
4. 성공 패널은 반영되고 실패 패널만 에러 표시되는지 확인한다.

### 10.4 워크스페이스 복원
1. split 레이아웃, 탭, 컬럼 설정, sync 설정을 구성한다.
2. Workspace 저장 후 앱을 새로고침한다.
3. Load 시 동일한 레이아웃/설정이 복원되는지 확인한다.

---

## 11) 완료 정의(Definition of Done)

아래를 모두 만족하면 MVP 완료:
- 섹션 10 테스트 항목 전부 통과
- `docs/spec-contracts.md`의 타입과 실제 구현 타입이 불일치하지 않음
- 인증 정보 디스크 저장 흔적이 없음
- 401/403 처리 흐름이 단일 규칙으로 동작
- 동기화 부분 실패 정책이 문서와 동일

---

## 12) 핵심 핸들러 계약(이 이름으로 구현 권장)

1. `handleCreateInstance(input)`
- 입력: `{ alias, baseUrl }`
- 검증 실패: 필드 에러 반환, 저장하지 않음
- 성공: `instances` 저장, `instanceRuntime[instanceId] = login_required`

2. `handleDeleteInstance(instanceId)`
- 동작: auth 제거 -> 관련 탭 닫기 -> favorites/presets 정리 -> 저장
- 성공 결과: UI에서 해당 인스턴스 참조가 모두 사라짐

3. `handleLogin(instanceId, email, password)`
- 성공: `authSession` 메모리 저장, 상태 `ok`
- 실패: 상태 유지, 모달 에러 표시

4. `handleOpenCollection(instanceId, collectionName)`
- 동작: 그룹/탭 생성 -> 기본 패널 상태 생성 -> `fetchPanelRecords(panelId)` 실행

5. `handleApplyFilter(panelId, filterQuery)`
- 동작: 패널 `filterQuery` 갱신, `page=1`, 조회
- 실패: 이전 정상 데이터 유지, 인라인 에러 표시

6. `handleToggleSort(panelId, field)`
- 동작: `none -> asc -> desc -> none` 순환, 조회

7. `handleSyncBroadcast(sourcePanelId)`
- 조건: `syncGroupId` 존재
- 동작: 대상 패널 병렬 전파(`allSettled`)
- 결과: 실패 패널만 에러 표시

8. `handleSaveWorkspace(name)`
- 동작: 현재 `layoutTree + groups + panelState` 직렬화 후 저장

9. `handleLoadWorkspace(workspaceId)`
- 동작: 로드 -> 패널별 데이터 조회 -> 렌더
- 예외: 인증 없는 인스턴스 패널은 `login_required` 상태로 유지
