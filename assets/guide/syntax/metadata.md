# 메타데이터 지시자

메타데이터 지시자는 블록(`{ }`) 안에 작성합니다. GUI 조작 결과도 이 형식으로 자동 삽입됩니다.

---

## @layout

노드의 위치와 크기를 지정합니다. GUI로 노드를 이동하거나 크기를 조절하면 자동으로 생성됩니다.

```grim
my-node {
    @layout {x: 100, y: 150}
}

my-node {
    @layout {x: 100, y: 150, w: 200, h: 80}
}
```

| 속성 | 설명 | 기본값 |
|------|------|--------|
| `x` | 좌측 상단 X 좌표 (px) | 자동 배치 |
| `y` | 좌측 상단 Y 좌표 (px) | 자동 배치 |
| `w` | 노드 너비 (px) | 콘텐츠에 맞게 자동 |
| `h` | 노드 높이 (px) | 콘텐츠에 맞게 자동 |
| `stack` | 겹친 모양으로 표시할 개수 | `1` |

중첩 노드에서는 **부모 노드의 좌측 상단을 (0, 0)으로 하는 상대 좌표**를 사용합니다.

```grim
parent {
    child1 {
        @layout {x: 0, y: 0}
    }
    child2 {
        @layout {x: 200, y: 0}
    }
}
```

`stack` 속성은 동일한 노드가 여러 개 있음을 시각적으로 표현할 때 사용합니다.

```grim
workers: "Worker 인스턴스" {
    @layout {stack: 3}
}
```

---

## @style

노드와 엣지의 시각적 스타일을 지정합니다.

```grim
my-node {
    @style {fill: blue-500, stroke: black, font-color: white, shadow: 3}
}
```

전체 속성 목록은 [스타일 속성](../reference/styles)을 참고하세요.

---

## @shape

노드의 모양을 지정합니다.

```grim
my-node {
    @shape circle
}

decision {
    @shape diamond
}
```

지원되는 모양: `rectangle`(기본값), `rounded`, `circle`, `diamond`, `cylinder`, `cloud`, `hexagon`, `parallelogram`

전체 목록과 설명은 [Shape 종류](../reference/shapes)를 참고하세요.

---

## @text

노드에 표시할 텍스트를 지정합니다. 콜론 문자열(`node: "hello"`)의 전체 표현입니다.

### plain 텍스트 (기본값)

```grim
my-node {
    @text hello world
}

# 따옴표 생략 가능, 줄 끝까지가 텍스트
# '#'이나 선행/후행 공백이 있으면 따옴표 사용
my-node {
    @text "hello world"
}

# 명시적으로 plain 타입 지정
my-node {
    @text[plain] hello
}
```

### 마크다운 텍스트

```grim
explanation {
    @text[markdown] {|
        # 제목
        - 항목 1
        - 항목 2
        **굵게** 및 *기울임* 지원
    |}
}
```

### LaTeX 텍스트

```grim
formula {
    @text[latex] {|
        \lim_{h \rightarrow 0 } \frac{f(x+h)-f(x)}{h}
    |}
}
```

> `{| ... |}` 블록은 여러 줄의 텍스트를 담을 때 사용합니다.

---

## @edge

엣지의 라우팅을 제어합니다. 스타일은 `@style`로 분리합니다.

```grim
A -> B {
    @edge {anchors: ["right", "left"], waypoints: [{x: 250, y: 150}], gap: {start: 1, end: 1}}
}
```

| 속성 | 설명 | 예시 값 |
|------|------|---------|
| `anchors` | 연결 지점 (from, to 순서) | `["right", "left"]` |
| `waypoints` | 경로의 꺾임점 좌표 목록 | `[{x: 100, y: 200}]` |
| `gap` | 노드 경계에서 선이 시작/끝나는 간격 (px) | `{start: 4, end: 4}` |

**anchor 값**: `top`, `bottom`, `left`, `right`, `center`

GUI에서 `Alt+Click`으로 웨이포인트를 추가/제거할 수 있습니다. 자세한 내용은 [단축키 & 인터랙션](../reference/keyboard)을 참고하세요.

---

## @icon

노드에 아이콘을 표시합니다. 외부 이미지 URL 또는 내장 아이콘 종류를 지정할 수 있습니다.

### 외부 이미지 아이콘

```grim
aws-backup {
    @icon { url: "https://icons.terrastruct.com/aws%2FStorage%2FAWS-Backup.svg" }
}

server {
    @icon { url: "https://example.com/icon.svg", position: top, size: 48 }
    @text 웹 서버
}
```

### 내장 아이콘

```grim
actor {
    @icon { kind: user }
    @text 사용자
}

ai {
    @icon { kind: bot }
    @text AI 에이전트
}

# position, size도 사용 가능
actor2 {
    @icon { kind: user, position: left, size: 32 }
    @text 사용자
}
```

| 속성 | 설명 | 기본값 |
|------|------|--------|
| `url` | 아이콘 이미지 URL | — |
| `kind` | 내장 아이콘 종류 (`user`, `bot`) | — |
| `position` | 텍스트 대비 아이콘 위치 | `top` |
| `size` | 아이콘 크기 (px) | `48` |

`url`과 `kind`는 상호 배타적입니다. 둘 중 하나만 지정하세요.

**kind 값**: `user` (사람 실루엣), `bot` (로봇)

**position 값**: `top`, `bottom`, `left`, `right`

---

## 지시자 배치 규칙

- 여러 지시자를 함께 사용할 수 있습니다.
- 지시자는 반드시 블록(`{ }`) 내부에 작성합니다.
- 콜론 문자열(`node: "text"`)과 블록 `@text`를 동시에 사용할 수 없습니다.

```grim
# 올바른 사용법 — 블록 내부에 지시자
server {
    @layout {x: 100, y: 100}
    @style {fill: blue-500}
    @shape rounded
    @text "API 서버"
}
```
