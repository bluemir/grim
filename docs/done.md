# grim 기획 검토

## 문법 및 데이터 모델

- [x] **1. 문법 정의 완성**
  - 노드 텍스트 표기 방식 통일/우선순위 정의 (콜론 문자열 vs `@text`)
  - 엣지 라벨: 인라인(`A -> B: label`) vs 블록(`@text`) 공존 규칙
  - 노드 ID 규칙 (허용 문자, 한글/공백/특수문자 처리)
- [x] **2. `@style` 속성 범위 정의**
  - 노드 스타일 속성 목록 (fill, stroke, shadow, font-size, opacity, border-width 등)
  - 엣지 스타일 속성 목록 (색상, 대시, 화살표 모양 등)
- [x] **3. `@shape` 종류 정의**
  - 지원할 shape 목록 (rectangle, circle, diamond, cylinder, cloud 등)

## 레이아웃 및 라우팅

- [x] **4. 자동 레이아웃(그리드 패킹) 상세 동작**
  - 그리드 셀 크기 결정 방식 (동적/고정)
  - 중첩 그룹 내부 미고정 노드 배치 규칙
  - 미고정 노드 배치 순서 (코드 순서? 연결 관계?)
- [x] **5. 엣지 라우팅 기본 동작**
  - `@edge` 없는 엣지의 기본 라우팅 방식
  - anchor 기본값 및 지원 목록
  - `gap` 단위와 의미

## 동기화 및 에러 처리

- [x] **6. 양방향 동기화 충돌 처리**
  - 코드 편집과 GUI 드래그 동시 발생 시 처리
  - 노드 삭제 시 관련 주석 처리
  - 문법 오류 상태에서의 GUI 동작

## 콘텐츠 렌더링

- [x] **7. Markdown/LaTeX 텍스트 블록**
  - 사용 가능 위치 (모든 노드 vs 전용 노드)
  - 렌더링 방식 (WASM SVG 변환 vs JS foreignObject)
- [x] **8. `@icon` 동작**
  - CORS 처리 방식
  - 아이콘+텍스트 동시 존재 시 레이아웃

## 프로젝트/운영

- [x] **9. 파일 구조 및 확장성**
  - 단일 파일 vs 멀티 파일(import/include)
  - 대규모 다이어그램 성능 한계
- [x] **10. 내보내기/공유**
  - 지원 포맷 (SVG, PNG, PDF 등)
  - URL 공유/임베딩 방식


## 기타
- [x] AST 구조체 정의 — Node, Edge, Metadata(@layout, @style, @shape, @text, @edge, @icon) 모델
- [x] 파서 구현 — .grim 텍스트 → AST
- [x] SVG 렌더러 — AST → SVG 문자열 (우선 rectangle + 텍스트만)
- [x] 연결 — grimRender()에서 파서 → 렌더러 호출
- [x] 이전 내용 보존
	- url 에 base64 인코딩해서 넣기
	- `http://localhost:8080/editor?data={base64_encoded_data}`
- [x] 마우스로 사각형 이동
- [x] edge 추가
- [x] 줌인 아웃
- [x] 화면 스크롤
	- wasd
- [x] 중첩 요소
- [x] 선언이 안되어 있는 node를 마우스로 옯기면 이동
- [x] 암시적 노드가 드래그 안됨
- [x] 24px 이나 16px 격자 grid 에 스냅
	- alt를 누르면 무시
	- js 단에만 적용하면 될듯
	- block 내부에는 블록 내부에서 다시 그리드가 있는것 처럼
- [x] 코드 편집 창에서 cmd+s, ctrl+s 막기
- [x] bug: @style {fill: blue-gray-200, font-color: white} 이렇게 지정했음에도, 배경색이 검은색으로 나옴
- [x] Shape 렌더링 — circle, diamond, cloud, cylinder, hexagon, parallelogram, rounded
- [x] 엣지 waypoints — Alt+Click으로 꺾임점 추가/제거
- [x] 엣지 anchor 지정 — top, bottom, left, right, center
- [x] Material Design 색상을 렌더러에 통합
- [x] bug: diamond 모양은 드래그로 위치를 바꿀수 없음
- [x] fixed 가 아닌 것은 fixed 와 거리를 두고 배치
	- fixed 중 가장 아래좌표에서 부터 배치 시작
	- 겹치지 않게 하기 위함.
- [x] code 포맷과 형식을 예쁘게 하는 fmt 기능이 필요함.
- [x] 역방향 화살표 지원 `A <- B`
- [x] 양방향 화살표 지원 `A <-> B`
- [x] 편집창에서 좌측 textarea 영역만 스크롤 우측은 주어진 크기에 꽉 차게
- [x] `@shape` 에 user , bot 추가 → `@icon {kind: user}` / `@icon {kind: bot}`으로 이전됨
- [x] access log 에서 특정 endpoint 는 query 를 찍지 않기
	- editor, viewer 는 data query 를 찍기 않도록
	- 실제 데이터라 너무 긴데, 서버에게 의미는 없음.
		- 아니면 데이터를 `#data={}` 로 담아야 할수도.
- [x] 어떤 노드가 같은게 여러개 있음 표시하기 위해 같은 모양을 뒤에 여러개 그려주는 기능
	- `@layout {stack: 3}` : 3개가 겹쳐진듯 보이게
- [x] 줌을 svg 의 1픽셀이 4px 나 8px 단위, 즉 400% 혹은 800% 까지 지원하기
	- 지금은 크기가 큰 svg의 경우 100% 위치가 이미 축소되어 있는 위치라 불편함.
	- svg 1px = html 1px 인 상태가 100% 가 되도록 변경
- [x] bug: 휠이 zoom in만 되고 zoom out 이 안됨
- [x] editor 와 viewer 는 화면이 좌우 여백 없이 꽉 차야 함.
- [x] 자식 노드에서 0,0 으로 지정하면, 부모의 헤더 바로 아래에 나와야 함.
