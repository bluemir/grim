# ADR-001: State Pattern 기반 2-Phase Parser

## Status
Accepted

## Date
2026-03-13

## Context
`.grim` 텍스트를 AST로 변환하는 파서가 필요하다. WASM 파이프라인(`textarea → grimRender → SVG`)이 연결된 상태에서 파서는 코어 엔진의 첫 번째 처리 단계이다.

파서 구현 방식으로 다음을 검토했다:
1. **재귀 하강 파서 (Recursive Descent)** — 직관적이지만 문법 변경 시 함수 간 결합도가 높아 수정 범위가 넓어짐
2. **Parser combinator** — 외부 라이브러리 의존 또는 높은 추상화 필요 (WASM 호환성 위험)
3. **State Pattern** — 상태별 처리 로직이 분리되어 개별 상태 교체/추가가 용이

`.grim` 문법은 점진적으로 확장될 예정이며(멀티 파일 import, 새로운 메타데이터 타입 등), 구현 변경의 용이성이 핵심 요구사항이다.

## Decision
**Lexer와 Parser 모두 State Pattern으로 구현하는 2-phase 아키텍처를 채택한다.**

### Phase 1: Lexer (rune → Token)
- `LexState` 인터페이스: `Handle(l *Lexer, ch rune) (LexState, error)`
- 상태: `lexDefault`, `lexIdent`, `lexString`, `lexComment`, `lexNumber`, `lexRawBlock`
- rune 단위로 소비하여 토큰 스트림 생성

### Phase 2: Parser (Token → AST)
- `ParseState` 인터페이스: `Handle(p *Parser, tok Token) (ParseState, error)`
- 상태: `stateTopLevel`, `stateInPath`, `stateAfterIdent`, `stateEdgeTarget`, `stateAfterColon`, `stateMetaKey`, `stateTextMeta`, `stateObjectMeta` 등
- 토큰 단위로 소비하며 AST 구축

### 설계 원칙
- **Permissive lexing**: 인식하지 못하는 문자도 토큰으로 emit (raw block 내 markdown/latex 지원)
- **Best-effort parsing**: 에러 발생 시 복구를 시도하고 부분 AST를 항상 반환 (GUI 편집 중 일시적 문법 오류에서 화면이 깨지지 않도록)
- **Block stack**: `{` / `}` 중첩을 스택으로 관리하여 nesting 지원
- **JSON-like object parsing**: `@layout`, `@style`, `@edge`, `@icon`의 속성값을 단순 JSON 문법으로 파싱
- **Raw block**: `@text[format] {| ... |}`는 전용 구분자를 사용하여 내부에서 `{`, `}` 가 자유롭게 등장 가능 (brace depth 추적 불필요)
- **표준 라이브러리만 사용**: WASM 호환성을 위해 `strings`, `fmt`, `strconv`, `unicode`만 사용

### 파일 구조
```
internal/parser/
  errors.go       -- ParseError, MultiParseError
  ast.go          -- AST 타입 (Document, NodeDecl, EdgeDecl, 메타데이터 등)
  token.go        -- Token 타입 + Lexer (LexState 기반)
  state.go        -- ParseState 인터페이스 + 구체 상태들
  parser.go       -- Parser 구조체, Parse() 진입점, JSON 헬퍼
  token_test.go   -- Lexer 단위 테스트 (91개)
  parser_test.go  -- Parser 통합 테스트 (105개)
```

## Consequences

### 장점
- **확장 용이**: 새 토큰 타입이나 문법 규칙 추가 시 새 State 구조체만 추가하면 됨
- **테스트 용이**: 각 State를 독립적으로 테스트 가능
- **에러 복구**: State 전이를 통해 에러 후 자연스럽게 상위 상태로 복귀
- **일관된 구조**: Lexer와 Parser가 동일한 패턴이므로 코드 이해 비용이 낮음

### 단점
- State 구조체 수가 많아질 수 있음 (현재 Lexer 6개, Parser 14개)
- State 간 데이터 전달을 위한 Parser 필드(`pathBuf`, `edgeFrom` 등)가 존재

### 제약사항
- 암시적 노드 해석(resolve)은 파서가 아닌 별도 semantic 단계에서 처리해야 함
- `A -> B: Read`에서 `Read`는 라벨(텍스트)이지 노드 참조가 아님
