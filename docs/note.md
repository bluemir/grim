# grim

"코드로 구조화하고, 마우스로 완성하는 완벽한 양방향 다이어그램 툴"

## Overview


- 배경: 기존 Diagram-as-Code 툴(D2, Mermaid 등)은 빠른 작성이 가능하지만, 오토 레이아웃 엔진의 한계로 인해 디테일한 위치 조정이 불가능함. 반면 GUI 툴(Draw.io 등)은 초기 구축이 번거롭고 버전 관리(Git)가 어려움.
- 목표: D2 스타일의 직관적인 문법을 차용하면서도, 사용자가 마우스로 조정한 레이아웃(좌표) 데이터가 코드에 실시간으로 동기화되는 '양방향 동기화(Two-way Sync)' 웹 기반 다이어그램 도구 개발.
- 철학: "논리적 구조는 코드로, 시각적 배치는 마우스로." 서버가 필요 없는 100% 브라우저(WASM) 구동.
- 파일 구조: v1은 단일 `.grim` 파일. 멀티 파일(import)은 추후 검토 (지원 시 번들링 포맷 필요).
- 내보내기/공유:
  - 다운로드: SVG, PNG
  - 공유: `/view?data={base64}` URL로 읽기 전용 뷰 제공 (선택적)
  - 임베딩: view URL을 `<iframe>`으로 삽입 가능 (view URL의 부산물)
  - PDF는 v1 미지원


## 핵심 기능 (Core Features)
###  D2 서브셋 기반의 직관적 문법
평면적 노드 선언 및 엣지 연결 (A -> B).
{ } 블록을 활용한 노드 중첩(Nesting) 지원.

### 메타데이터 주석 기반 양방향 동기화
GUI에서 변경된 레이아웃(좌표), 스타일, 엣지 라우팅 정보는 코드 원본을 훼손하지 않고 `@layout`, `@style`, `@edge` 형태의 주석으로 노드 선언부 바로 위에 주입됨.
암시적 선언 vs 명시적 선언: 코드로만 연결(A -> B)해둔 상태에서 GUI로 노드를 움직이면, 시스템이 자동으로 해당 그룹 내에 노드를 명시적으로 선언하고 레이아웃 주석을 추가함.

### 혼합 레이아웃 (Mixed Layout)
고정 노드와 미고정 노드는 **분리된 영역**에 배치된다. (고정 노드는 사용자가 마우스로 옮기려는 의도가 있으므로 자동 배치와 독립)

**고정 노드** (`@layout` 있음):
- 절대/상대 좌표에 정확히 배치
- 중첩 노드의 경우 부모 노드의 좌측 상단을 (0,0)으로 하는 상대 좌표

**미고정 노드** (`@layout` 없음) — 12-column 그리드 자동 배치:
- CSS Grid `auto-flow: dense` 방식
- 그리드 전체 폭: 600px ~ 1920px (뷰포트 기반)
- 노드 콘텐츠 크기에 따라 1col~12col 점유 (텍스트 길이, shape 최소 크기 기반 자동 계산)
- 배치 순서: 코드 선언 순서 (위→아래), 흐름 방향: 왼쪽→오른쪽, 넘치면 다음 줄
- 중첩 그룹 내부의 미고정 노드도 동일 규칙 적용 (부모 그룹 폭 기준)

### 엣지 라우팅 및 픽셀 컨트롤
- **기본 라우팅**: 직선(straight line) — 두 노드 중심 간 직선이 노드 경계와 만나는 점을 anchor로 자동 계산
- **gap**: 노드 경계로부터 선이 시작/끝나는 간격 (px, 기본값: 4px)
- **웨이포인트**: 엣지 위에 Ctrl + Click으로 꺾임 지점을 자유롭게 추가/제거 가능
- **anchor 지정**: `@edge`에서 `anchors: ["right", "left"]` 등으로 수동 지정 가능 (top, bottom, left, right, center)
- `@edge`는 라우팅 전용 (anchors, waypoints, gap), 스타일은 `@style`로 분리


## 시스템 아키텍처 (Architecture)

- **grim**은 극한의 성능과 이식성을 위해 코어 연산과 프론트엔드 렌더링을 철저히 분리합니다.
	- Core Engine (Go -> WASM): * 텍스트 파싱, AST(추상 구문 트리) 생성, 그리드 자동 정렬 계산, 최종 SVG 문자열 생성 담당.
	- 브라우저 내에서 C/C++ 급의 성능으로 동작하며 서버 요청(API) 불필요.
- Frontend View (Vanilla JS / Lit-html):
	- 가벼운 `<textarea>` 에디터와 SVG DOM 제어.
	- 이벤트 위임(Event Delegation)을 통해 SVG의 상호작용(드래그, 클릭) 처리.
- Data Binding (SVG 커스텀 속성):
	- WASM이 뱉어내는 SVG 태그에 data-id, data-line(텍스트 에디터의 줄 번호) 등을 심어, JS가 텍스트 코드를 업데이트할 때 정규식 탐색 없이 즉시 해당 줄을 찾아 수정(O(1) 성능).


## UX / UI 상호작용 설계 (Interactions)

- 60fps 드래그 앤 드롭: * 드래그 중에는 WASM 호출 없이 JS가 DOM(SVG transform)만 직접 제어하여 부드러운 조작감 제공.
	- 마우스 드롭(Mouse Up) 시점에만 JS가 `<textarea>`의 특정 줄을 업데이트하고 WASM 동기화 수행.
- 에디터 커서 튐 방지: 코드 업데이트 전후로 selectionStart/End를 캡처하여 유저의 타이핑 경험 보호.
- 투명 히트박스(Hitbox): 얇은 선(Edge)을 쉽게 클릭할 수 있도록 눈에 보이지 않는 두꺼운 <path>를 겹쳐 렌더링.
	- wasm 에서는 렌더하지 않지만, JS에서 두꺼운 path 를 겹치치도록 추가

### 양방향 동기화 충돌 처리
- **코드 편집**: debounce (예: 300ms) 후 WASM 파싱 수행
- **드래그 중 충돌 방지**: 드래그 중에는 JS DOM만 제어, WASM 호출 없음. 드롭 시점에 **data-id 기반**으로 최신 텍스트에서 해당 노드의 `@layout` 줄을 찾아 업데이트
- **드롭 실패 처리**: 노드가 코드에서 삭제되었거나 파싱 에러 상태이면 드래그 무효화, 최신 유효 상태로 재렌더링
- **문법 오류 시 GUI**: 마지막 유효 파싱 결과의 SVG를 유지하고, 에러 표시만 추가 (타이핑 중 일시적 오류에 화면이 사라지지 않도록)


## 문법 규칙 (Syntax Rules)

### 노드 ID 규칙
- RFC 1123 DNS Label 기반, 대소문자 구분(case-sensitive) 확장
- 정규식: `[a-zA-Z][a-zA-Z0-9_-]*`
  - 영문으로 시작
  - 이후 영문, 숫자, 하이픈(`-`), 언더스코어(`_`) 허용
- 점(`.`)은 경로 구분자 전용 (ID에 사용 불가) — 예: `parent.child.grandchild`
- 한글/공백은 v1에서 미지원

### 텍스트(라벨) 규칙
- 콜론 문자열은 블록 `@text`의 축약형
  - `node: "hello"` === `node: { @text "hello" }`
  - `A -> B: "label"` === `A -> B: { @text "label" }`
- 축약형과 블록 `@text`는 동시에 존재할 수 없음 (문법 에러)
- `@text` 타입 지정은 브라켓 표기법 사용:
  - `@text hello world` — 따옴표 생략 가능, 줄 끝(또는 `#` 주석)까지가 텍스트
  - `@text "hello world"` — 따옴표 명시 (`#`이나 선행/후행 공백 포함 시 사용)
  - `@text[plain] hello` — plain 타입 명시 (기본값과 동일)
  - `@text[markdown] {| ... |}` — 마크다운
  - `@text[latex] {| ... |}` — LaTeX
- 모든 노드에서 모든 텍스트 타입 사용 가능
- 렌더링 방식 (D2와 동일한 접근):
  - plain → 순수 SVG `<text>` 요소 (최대 호환성)
  - markdown → Go(goldmark)에서 HTML 변환 후 SVG `<foreignObject>`에 삽입 (CLI 확장에도 유리)
  - latex → JS(KaTeX)에서 변환 후 `<foreignObject>`에 삽입 (Go 측 변환은 추후 조사)
  - WASM은 plain/markdown을 완성된 SVG로 출력, latex는 영역만 확보하고 JS가 채움
- Export:
  - PNG → canvas 래스터화 (foreignObject 호환 문제 회피)
  - SVG → foreignObject 포함 상태로 저장

### 색상 표기
- CSS 호환 형식 지원: `#123456`, `rgb(r,g,b)`, `rgba(r,g,b,a)`
- Material Design 사전 정의 색상 팔레트 (`assets/src/css/material/color/` 기반):
  - 19개 계열: `red`, `pink`, `purple`, `deep-purple`, `indigo`, `blue`, `light-blue`, `cyan`, `teal`, `green`, `light-green`, `lime`, `yellow`, `amber`, `orange`, `deep-orange`, `brown`, `gray`, `blue-gray`
  - 단계: `50`, `100`~`900` (예: `gray-100`, `blue-500`)
  - alt 변형 지원 (예: `blue-alt-200`, `blue-alt-700`)
- 편의 색상: `black`, `white`, `transparent`

### 아이콘 규칙 (`@icon`)
- `@icon { url: "https://...", position: top, size: 48 }`
- 속성:
  - `url`: 아이콘 URL (필수)
  - `position`: 텍스트 대비 아이콘 위치 — `top`(기본), `left`, `right`, `bottom`
  - `size`: 아이콘 크기 (px, 기본값: 48)
- 렌더링: SVG `<image href="URL">` 사용
- PNG export 시 외부 URL은 fetch → base64 인라인 변환 시도, 실패 시 아이콘 생략

### Shape 규칙 (`@shape`)
- `@shape` 뒤에 shape 종류를 단일 값으로 지정
- 지원 목록: `rectangle` (기본값), `rounded`, `circle`, `diamond`, `cylinder`, `cloud`, `hexagon`, `parallelogram`
- 예: `@shape diamond`

### 레이아웃 규칙 (`@layout`)
- 위치와 크기를 포함하는 기하 정보 (GUI 조작으로 자동 주입됨)
- `@layout {x: 100, y: 150}` — 위치만 지정 (크기는 콘텐츠에 맞게 자동)
- `@layout {x: 100, y: 150, w: 200, h: 80}` — 위치 + 크기 지정
- 중첩 노드의 경우 부모 노드의 좌측 상단을 (0,0)으로 하는 상대 좌표

### 스타일 규칙 (`@style`)
노드와 엣지 모두 `@style`로 스타일을 지정한다. `@edge`는 라우팅 전용.

**노드 스타일 속성:**
| 속성 | 설명 | 예시 값 |
|---|---|---|
| `fill` | 배경색 | `blue`, `#3498db`, `transparent` |
| `stroke` | 테두리 색 | `black`, `#333` |
| `stroke-width` | 테두리 두께 (px) | `2` |
| `dash` | 테두리 점선 패턴 | `5,3` |
| `shadow` | 그림자 크기 (px, 0이면 없음) | `3` |
| `opacity` | 투명도 (0~1) | `0.5` |
| `font-size` | 텍스트 크기 (px) | `14` |
| `font-color` | 텍스트 색 | `white`, `#fff` |
| `border-radius` | 모서리 둥글기 (px) | `8` |

**엣지 스타일 속성:**
| 속성 | 설명 | 예시 값 |
|---|---|---|
| `stroke` | 선 색 | `red`, `#333` |
| `stroke-width` | 선 두께 (px) | `2` |
| `dash` | 점선 패턴 | `5,3` |
| `arrow` | 화살표 모양 | `triangle`, `diamond`, `none` |

## 데이터 모델 및 문법 예시 (Syntax Example)


```grim

network: {
  @layout {x: 0, y: 0, w: 600, h: 400}
  @style  {fill: blue, stroke: black, shadow: 3}

  frontend: {
	@layout {x: 100, y: 150}
  	@style  {fill: blue, stroke: black}
  }

  backend: {
	@layout {x: 400, y: 150}
  }

  frontend -> backend: {
	@edge {anchors: ["right", "left"], waypoints: [{x: 250, y: 150}], gap: {start: 1, end : 1}}
	@text API Call
  }
}

# 고정되지 않은 database 노드는 빈 공간에 자동 배치됨
frontend -> database: Read

explanation: {
	@text[markdown] {|
		# I can do headers
		- lists
		- lists

		And other normal markdown stuff
	|}
}

formula: {
	@text[latex] {|
		\lim_{h \rightarrow 0 } \frac{f(x+h)-f(x)}{h}
	|}
}

icon-node: {
	@icon { url: "https://icons.terrastruct.com/aws%2FStorage%2FAWS-Backup.svg"}
}

pg: {
	@shape cloud
	@text PostgreSQL
}

my-node: {
	inside-node: {
		inside-sq-node: "hello"
	}
}

my-node.inside-node.inside-sq-node -> pg

# comment


```
