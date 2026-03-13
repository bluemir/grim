# ADR-003: 드래그 레이아웃 업데이트를 AST 기반 WASM 파이프라인으로 교체

## Status
Accepted

## Date
2026-03-14

## Context
노드를 드래그하면 JS가 소스 텍스트에서 해당 노드의 `@layout`을 직접 갱신해야 한다.
기존 구현(`updateNodeLayout`)은 소스 텍스트를 정규식/문자열 조작으로 처리했으며, 다음 문제가 있었다:

- **브레이스 깊이 추적 오류**: 부모 노드를 드래그할 때 자식 블록 안의 `@layout`을 잘못 수정하는 버그
- **파편화된 케이스 처리**: 암시적 노드 / 블록 있음 / 블록 없음 / 기존 `@layout` 존재 여부를 각각 분기하는 63줄 함수
- **테스트 불가**: JS 문자열 조작 로직은 단위 테스트 작성이 어려움
- **포맷 불일치**: 삽입된 코드가 `grimFormat` 결과와 다른 형식을 가질 수 있음

이미 `Parse`와 `Format` 함수가 존재하므로, 모든 소스 조작을 AST 수준에서 처리할 수 있다.

## Decision
**JS의 문자열 조작 대신 `Parse → AST 조작 → Format` 파이프라인을 사용한다.**

### 구현

**`internal/parser/layout.go`** — 새 파일:
- `SetLayout(doc *Document, nodeID string, x, y int64)` — 공개 API
- `findNodeByPath(stmts, segments)` — flat dotted(`parent.child {}`)와 nested block(`parent { child {} }`) 두 패턴 모두 재귀 탐색
- `createNodeByPath(doc, segments)` — 소스에 선언이 없는 암시적 노드를 최심 조상 기준으로 중첩 블록 형식으로 삽입
- `updateLayoutMeta(node, x, y)` — 기존 `@layout`의 x/y만 교체(w/h 등 기존 키 보존), 없으면 prepend

**`wasm/wasm.go`** — `grimSetLayout` 추가:
```
grimSetLayout(code, nodeID, x, y) → string
```
Parse 에러 시 원본 코드 반환 (`grimFormat`과 동일한 안전 정책).

**`assets/html-templates/editor.html`** — `mouseup` 핸들러 교체:
```js
// 전
updateNodeLayout(id, line, newX, newY);
// 후
editor.value = grimSetLayout(editor.value, id, newX, newY);
```
`data-line` 의존성 제거, `updateNodeLayout` 함수 전체 삭제.

### 타입 선택
x, y를 `int64`로 저장. `parseJSONValue`가 정수를 `int64`로 저장하고 `writeJSONValue`가 소수점 없이 출력하므로 완전한 round-trip이 보장된다.

## Consequences

### 장점
- **버그 수정**: 부모 드래그 시 자식 `@layout`이 변경되지 않음 — JS 브레이스 깊이 추적 로직 불필요
- **테스트 가능**: Go 단위 테스트 8개로 모든 케이스 검증
- **포맷 일관성**: 삽입/수정 결과가 항상 `grimFormat`과 동일한 형식
- **코드 감소**: JS 63줄 → 1줄 호출
- **암시적 노드 처리**: `data-line`이 필요 없어지므로 렌더러가 라인 정보를 SVG에 embed하지 않아도 됨 (향후 정리 가능)

### 단점
- 드래그 한 번에 Parse + Format이 추가로 실행됨 (mouseup 시점이므로 성능 영향 미미)

### 제약사항
- `grimSetLayout`은 소스 전체를 재파싱하므로, 파싱 불가능한 부분 입력 상태에서는 레이아웃이 반영되지 않음
