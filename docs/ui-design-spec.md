# Multi PocketBase UI - 디자인 명세 (MVP)

이 문서는 `docs/ui-spec.md` 기능을 구현할 때 필요한 시각/상호작용 기준을 정의한다.
기능 정책 충돌은 `docs/ui-spec.md`, 문서 권한 충돌은 `docs/docs-index.md`를 따른다.

실제 토큰 파일: `styles/tokens.css`

## 1) 디자인 목표

1. 고밀도 데이터 UI에서도 가독성을 유지한다.
2. Explorer / Canvas / Inspector의 역할을 명확히 구분한다.
3. 인증 만료/오류 상태를 즉시 인지할 수 있게 한다.

## 2) 스타일 방향

- 테마: 라이트 1종(MVP)
- 톤: 저채도 중립 + 명확한 포커스 컬러
- 밀도: 컴팩트 우선

## 3) 토큰

### 3.1 컬러

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
1. 본문 텍스트는 `--text-primary`
2. 보조 정보는 `--text-secondary`/`--text-muted`
3. 주요 액션만 `--accent`
4. 상태 표현은 배경(`*-soft`) + 텍스트(`danger/warning/success`) 조합

### 3.2 타이포그래피

- 기본 폰트: `"Pretendard Variable", "Noto Sans KR", sans-serif`
- 숫자/코드 폰트: `"JetBrains Mono", "SFMono-Regular", monospace`

타입 스케일:
- 12/18
- 13/20 (테이블 기본)
- 14/22 (본문 기본)
- 16/24 (섹션 헤더)

### 3.3 간격/크기

4pt 그리드:
- 4, 8, 12, 16, 20, 24px

고정 높이:
- Top Bar 48px
- 입력/버튼 32px
- 탭 34px
- 테이블 헤더 34px
- 테이블 바디 32px

## 4) 레이아웃

- 최소 창 크기: `1280 x 800`
- 권장 창 크기: `1440 x 900` 이상

영역 폭 기준:
- Left Explorer: 260px (min 220, max 360)
- Right Inspector: 320px (min 280, max 420)
- Center Canvas: 나머지 전부

정적 프리뷰는 2열(Explorer + Canvas)로 축약 가능하다.

## 5) 컴포넌트 규칙

### 5.1 Top Bar
- 배경 `--bg-panel`
- 하단 경계 `--divider`
- 주요 액션과 일반 액션을 시각적으로 구분

### 5.2 버튼
- `primary`, `secondary`, `ghost` 3종
- hover/active/disabled/focus 상태를 모두 제공
- focus는 `--focus-ring`으로 명확히 표시

### 5.3 입력/셀렉트
- 기본 높이 32px
- focus 시 `--accent` 강조
- 에러 시 `--danger` 경계 + 보조 문구

### 5.4 탭
- 활성 탭은 상단 accent 라인
- 비활성 탭은 보조 텍스트
- 닫기 아이콘은 hover 배경 제공

### 5.5 Explorer 리스트
- 항목 높이 28px
- 선택 항목은 좌측 accent 바 + 선택 배경
- 즐겨찾기 활성은 warning 색상

### 5.6 테이블
- Header/Subtle 배경, Body 교차 행 가독성
- 행 hover/selected 상태 구분
- 숫자 컬럼 우측 정렬
- Pinned 컬럼은 경계 그림자로 분리

### 5.7 Inspector
- 탭 영역/내용 영역 명확히 분리
- JSON/객체 값은 코드 폰트 + subtle 배경

### 5.8 모달
- 배경 딤 + 본문 패널 대비 확보
- header/body/footer 고정 구조
- 키보드 Esc 닫기와 포커스 트랩 지원

### 5.9 배지/토스트
- 인증 만료 배지: warning 계열
- 토스트: 우상단 스택, 자동 닫힘 4초

## 6) 상태별 시각 규칙

### 6.1 로딩
- skeleton 우선
- 기존 데이터가 있으면 overlay spinner 사용

### 6.2 빈 상태
- 아이콘 + 제목 + 설명 + 보조 액션 1개

### 6.3 에러 상태
- 필터 오류: 입력 인접 영역에 직접 표시
- 네트워크 오류: 패널 상단 배너 + 재시도
- 인증 만료: 패널 상단 배지 + 재로그인 CTA

## 7) 인터랙션/모션

모션 토큰:
- fast 120ms
- normal 180ms
- slow 240ms
- easing `cubic-bezier(0.2, 0, 0, 1)`

규칙:
1. hover/focus는 fast
2. 탭/패널 전환은 normal
3. 모달은 opacity + scale 전환
4. Split 리사이즈는 실시간 반영(애니메이션 없음)

## 8) 접근성

필수 기준:
- 본문 대비 4.5:1 이상
- 아이콘 버튼 `aria-label` 필수
- 키보드 접근 동등성 제공

기본 키보드 규칙:
- 컬렉션 선택 이동: Up/Down
- 열기: Enter
- 탭 이동: `Ctrl+Tab`, `Ctrl+Shift+Tab`
- 패널 닫기: `Ctrl+W`
- 모달 닫기: Esc

## 9) 반응형 (Desktop)

- 1600px 이상: 기본 폭 유지
- 1280~1599px: Inspector 폭 축소
- 1279px 이하: 비권장 해상도 안내
