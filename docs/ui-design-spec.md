# Multi PocketBase UI - 디자인 상세 명세 (MVP)

이 문서는 `docs/ui-spec.md`의 기능을 구현할 때 필요한 시각/상호작용 기준이다.
기능 명세와 충돌 시 기능 명세를 우선하고, 디자인 문서는 즉시 수정한다.

실제 토큰 파일:
- `styles/tokens.css`

---

## 1) 디자인 목표

1. 많은 컬럼을 다뤄도 눈 피로가 낮은 고밀도 데이터 UI를 제공한다.
2. 탐색(Explorer), 작업(Canvas), 상세(Inspector)의 역할을 색/경계/간격으로 분명히 구분한다.
3. 경고/오류/인증 만료 상태를 즉시 인지할 수 있게 한다.

---

## 2) 기본 스타일 방향

- 기본 테마: 라이트 테마 1종 (MVP)
- 톤: 저채도 중립 + 명확한 포커스 컬러
- 시각 밀도: 컴팩트(기본), 넉넉한 간격보다 정보량 우선

---

## 3) 디자인 토큰

## 3.1 컬러 토큰

```css
:root {
  --bg-app: #f4f6f8;
  --bg-panel: #ffffff;
  --bg-subtle: #f8fafc;
  --bg-hover: #eef2f7;
  --bg-selected: #e8f0fe;

  --border-default: #d8dee8;
  --border-strong: #b8c2d1;
  --divider: #e5eaf1;

  --text-primary: #1f2937;
  --text-secondary: #4b5563;
  --text-muted: #6b7280;
  --text-inverse: #ffffff;

  --accent: #2563eb;
  --accent-hover: #1d4ed8;
  --accent-soft: #dbeafe;

  --success: #15803d;
  --warning: #b45309;
  --danger: #b91c1c;
  --danger-soft: #fee2e2;
  --warning-soft: #fef3c7;
  --success-soft: #dcfce7;

  --focus-ring: #60a5fa;
}
```

컬러 사용 규칙:
1. 본문 텍스트는 `--text-primary`만 사용
2. 보조 정보는 `--text-secondary` 또는 `--text-muted`만 사용
3. 클릭 가능 주요 액션만 `--accent`
4. 오류/경고 상태는 배경(`*-soft`) + 텍스트(`danger/warning`) 조합 사용

## 3.2 타이포그래피 토큰

- 기본 폰트: `"Pretendard Variable", "Noto Sans KR", sans-serif`
- 숫자/코드 폰트: `"JetBrains Mono", "SFMono-Regular", monospace`

타입 스케일:
- `font-12`: 12px / 18px
- `font-13`: 13px / 20px (테이블 기본)
- `font-14`: 14px / 22px (본문 기본)
- `font-16`: 16px / 24px (섹션 헤더)

폰트 굵기:
- `regular`: 400
- `medium`: 500
- `semibold`: 600

## 3.3 간격/크기 토큰

4pt 그리드 사용:
- `space-1`: 4px
- `space-2`: 8px
- `space-3`: 12px
- `space-4`: 16px
- `space-5`: 20px
- `space-6`: 24px

반경/경계:
- `radius-sm`: 6px
- `radius-md`: 10px
- `radius-lg`: 14px
- `border-1`: 1px

높이 토큰:
- Top Bar: 48px
- 입력/버튼(기본): 32px
- 입력/버튼(컴팩트): 28px
- 탭 높이: 34px
- 테이블 헤더 행: 34px
- 테이블 바디 행: 32px

---

## 4) 레이아웃 규격

## 4.1 앱 최소 크기

- 최소 창 크기: `1280 x 800`
- 권장 창 크기: `1440 x 900` 이상

## 4.2 영역 폭

- Left Explorer: 기본 260px, 최소 220px, 최대 360px
- Right Inspector: 기본 320px, 최소 280px, 최대 420px
- Center Canvas: 잔여 폭 전체

정적 UI 프리뷰(`preview/*.html`) 기준:
- Right Inspector 영역을 사용하지 않는다.
- 2열 구조(Left Explorer + Center Canvas)에서 패널 내부 컨트롤을 강화한다.

## 4.3 패널 여백

- 앱 전체 패딩: 8px
- 패널 내부 패딩: 8px
- 섹션 간 간격: 8px

---

## 5) 컴포넌트 상세

## 5.1 Top Bar

스타일:
- 배경: `--bg-panel`
- 하단 경계: `1px solid --divider`
- 좌우 패딩: 12px

버튼 그룹 규칙:
- 주요 액션(`Save`, `Load`)은 기본 버튼
- 파괴 액션(없음)은 danger 버튼 미사용

## 5.2 버튼

크기:
- 기본: 32px 높이, 좌우 패딩 12px
- 컴팩트: 28px 높이, 좌우 패딩 10px

종류:
- `primary`: accent 배경
- `secondary`: panel 배경 + border
- `ghost`: 배경 없음, hover에서 subtle 배경

상태:
- hover: 배경 또는 보더 강화
- active: 1단계 어둡게
- disabled: opacity 0.45 + pointer-events none
- focus: `2px solid --focus-ring` 아웃라인

## 5.3 입력/셀렉트/검색

스타일:
- 높이 32px
- 배경 `--bg-panel`
- 보더 `--border-default`
- placeholder `--text-muted`

상태:
- focus: border `--accent`, focus ring 표시
- error: border `--danger`, 보조 텍스트 `--danger`

## 5.4 탭(Tab)

탭 바:
- 높이 34px
- 배경 `--bg-subtle`
- 보더 하단 `1px solid --divider`

탭 아이템:
- 기본: `text-secondary`
- 활성: `text-primary`, 배경 `--bg-panel`, 상단 2px `--accent`
- hover: `--bg-hover`

닫기 아이콘:
- 탭 우측 16px 아이콘
- hover 시 원형 배경 `--bg-hover`

## 5.5 Explorer 리스트

항목 높이:
- 컬렉션 항목 28px

상태:
- hover: `--bg-hover`
- selected: `--bg-selected` + 좌측 2px `--accent`

즐겨찾기 아이콘:
- 기본 `--text-muted`
- 활성 `--warning`

## 5.6 테이블

테이블 컨테이너:
- 배경 `--bg-panel`
- 보더 `1px solid --border-default`
- 반경 `radius-sm`

헤더:
- 높이 34px
- 배경 `--bg-subtle`
- 텍스트 `font-12 semibold`

바디:
- 행 높이 32px
- 홀수 행 배경 `--bg-panel`
- 짝수 행 배경 `#fcfdff`
- hover 행 배경 `--bg-hover`
- selected 행 배경 `--bg-selected`

셀 규칙:
- 좌우 패딩 10px
- 텍스트 overflow ellipsis
- 숫자 컬럼 우측 정렬

Pinned 컬럼:
- 좌측 고정 영역 배경 `--bg-panel`
- 경계 그림자 `inset -1px 0 0 var(--divider)`

## 5.6.1 필드(컬럼) 선택 UI (Panel Inline)

구성:
- 제목: `Fields`
- 선택 개수 배지: `N selected`
- 토글 버튼: `Choose Columns` / `Close Columns`
- 옵션: 체크박스 + 컬럼명 칩

배치:
- 필터/정렬/페이징 바 바로 아래
- 테이블 컨텐츠 바로 위
- 패널 내부 구분선(`--divider`)으로 분리

상태:
- 기본: 옵션 영역 닫힘
- 열림: 옵션 칩 wrap 배치
- 체크 해제: 해당 컬럼 즉시 숨김
- 최소 1개 컬럼은 항상 유지

## 5.7 Inspector

탭 영역과 내용 영역 분리:
- 탭 바 높이 34px
- 내용 패딩 12px

필드 행:
- label 30%, value 70%
- 줄 간격 8px

JSON 뷰:
- 코드 폰트 사용
- 배경 `--bg-subtle`
- 보더 `--border-default`
- 반경 `radius-sm`

## 5.8 모달

레이어:
- 오버레이 `rgba(15, 23, 42, 0.35)`
- 모달 배경 `--bg-panel`
- 너비 420px (로그인), 560px (워크스페이스 로드)

구성:
- header / body / footer 고정 구조
- footer 버튼 우측 정렬

## 5.9 배지/토스트

배지:
- 높이 20px, 폰트 12px
- `auth_expired`: 배경 `--warning-soft`, 텍스트 `--warning`

토스트:
- 우상단 스택
- 한 개 높이 44px
- 자동 닫힘 4초

---

## 6) 상태별 시각 규칙

## 6.1 로딩

- 데이터 영역에 skeleton 5행 표시
- 로딩 중 기존 데이터가 있으면 overlay spinner만 표시

## 6.2 빈 상태

구성:
- 아이콘(32px)
- 제목 14px semibold
- 설명 13px regular
- 보조 버튼 1개(예: 필터 초기화)

## 6.3 에러 상태

필터 에러:
- 입력 보더 danger
- 입력 하단 12px 에러 문구

네트워크 에러:
- 패널 상단 인라인 배너
- `재시도` 버튼 포함

인증 만료:
- 패널 상단 배지 + 로그인 CTA

---

## 7) 인터랙션/모션

모션 토큰:
- `fast`: 120ms
- `normal`: 180ms
- `slow`: 240ms
- easing: `cubic-bezier(0.2, 0, 0, 1)`

모션 규칙:
1. hover/focus 전환: `fast`
2. 탭/패널 전환: `normal`
3. 모달 open/close: opacity + scale(0.98 -> 1.00), `normal`
4. Split 리사이즈는 애니메이션 없이 실시간 반영

---

## 8) 접근성(A11y)

필수 기준:
- 본문 텍스트 대비 4.5:1 이상
- 아이콘 단독 버튼은 `aria-label` 필수
- 모든 주요 액션 키보드 접근 가능

키보드 규칙:
- Explorer 이동: Up/Down
- 컬렉션 열기: Enter
- 탭 이동: Ctrl+Tab / Ctrl+Shift+Tab
- 패널 닫기: Ctrl+W
- 모달 닫기: Esc

포커스 규칙:
- 키보드 포커스는 항상 가시적 아웃라인 표시
- 모달 오픈 시 포커스 트랩 적용

---

## 9) 반응형 규칙 (Desktop 범위)

- 1600px 이상: 기본 폭 유지
- 1280~1599px: Inspector 기본 폭을 280px로 축소
- 1279px 이하: 지원하지 않음 안내 배너 표시(기능 동작은 가능하되 레이아웃 깨짐 보장 안 함)

---

## 10) 구현 체크리스트

1. CSS 변수로 토큰을 먼저 선언했다.
2. 컴포넌트에서 하드코딩 색상/간격을 사용하지 않았다.
3. hover/focus/disabled/error 상태를 모두 구현했다.
4. 테이블 행/헤더 높이가 토큰과 일치한다.
5. `auth_expired`, `network_error`, `filter_invalid` 시각 표현이 구분된다.
6. 모달 포커스 트랩, Esc 닫기가 동작한다.
7. 1280px, 1440px, 1728px에서 레이아웃이 무너지지 않는다.
8. 패널 내 `Fields` 선택 UI에서 컬럼 표시/숨김이 즉시 반영된다.

---

## 11) 샘플 CSS 뼈대

```css
body {
  margin: 0;
  font-family: "Pretendard Variable", "Noto Sans KR", sans-serif;
  font-size: 14px;
  line-height: 22px;
  color: var(--text-primary);
  background: var(--bg-app);
}

.app-shell {
  display: grid;
  grid-template-rows: 48px 1fr;
  height: 100vh;
}

.content {
  display: grid;
  grid-template-columns: 260px 1fr 320px;
  gap: 8px;
  padding: 8px;
  min-width: 1280px;
}

.panel {
  background: var(--bg-panel);
  border: 1px solid var(--border-default);
  border-radius: 10px;
}
```

적용 방법:
```css
@import "../styles/tokens.css";
```
