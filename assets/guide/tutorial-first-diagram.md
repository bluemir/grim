# 튜토리얼: 첫 다이어그램 만들기

이 튜토리얼을 따라하면 grim의 기본 사용법을 배울 수 있습니다. 완성하면 간단한 웹 서비스 아키텍처 다이어그램이 만들어집니다.

---

## 1단계: 에디터 열기

grim 에디터 URL에 접속합니다. 좌측은 코드 에디터, 우측은 SVG 미리보기입니다.

---

## 2단계: 첫 노드 선언하기

좌측 에디터에 아래 코드를 입력합니다.

```grim
client
server
database
```

입력을 멈추면 우측에 세 개의 노드가 자동으로 그리드에 배치됩니다.

> 노드 ID는 영문 소문자로 시작하고, 영문/숫자/하이픈/언더스코어만 사용할 수 있습니다.

---

## 3단계: 엣지로 연결하기

노드 사이에 관계를 정의합니다.

```grim
client
server
database

client -> server
server -> database
```

`->` 로 방향 있는 화살표를 그립니다. 코드를 저장하면 두 개의 엣지가 생깁니다.

---

## 4단계: 라벨 추가하기

엣지와 노드에 설명을 붙입니다.

```grim
client: "웹 브라우저"
server: "API 서버"
database: "PostgreSQL"

client -> server: "HTTP 요청"
server -> database: "SQL 쿼리"
```

---

## 5단계: 마우스로 위치 조정하기

우측 미리보기에서 노드를 드래그하여 위치를 조정합니다.

- 노드를 클릭하면 **선택 상태**가 되고, 코드 에디터의 커서가 해당 노드 선언 위치로 이동합니다.
- 드래그 후 마우스를 떼면 위치 정보가 `@layout`으로 코드에 자동 추가됩니다.

코드가 아래처럼 변경됩니다:

```grim
client: "웹 브라우저" {
    @layout {x: 0, y: 0}
}
server: "API 서버" {
    @layout {x: 200, y: 0}
}
database: "PostgreSQL" {
    @layout {x: 400, y: 0}
}

client -> server: "HTTP 요청"
server -> database: "SQL 쿼리"
```

---

## 6단계: 스타일 적용하기

노드에 색상과 모양을 지정합니다.

```grim
client {
    @layout {x: 0, y: 0}
    @style {fill: blue-500, font-color: white}
    @text "웹 브라우저"
}

server {
    @layout {x: 200, y: 0}
    @style {fill: green-500, font-color: white}
    @text "API 서버"
}

database {
    @layout {x: 400, y: 0}
    @shape cylinder
    @style {fill: orange-500, font-color: white}
    @text "PostgreSQL"
}

client -> server: "HTTP 요청"
server -> database: "SQL 쿼리"
```

> Material Design 색상 팔레트를 사용합니다. `blue-500`, `green-200` 등의 형식으로 지정합니다. 전체 목록은 [스타일 속성](reference/styles)을 참고하세요.

---

## 7단계: Format으로 코드 정리하기

좌측 상단의 **Format** 버튼을 클릭하거나 `Shift+Alt+F`를 누릅니다. 들여쓰기와 공백이 일관되게 정리됩니다.

---

## 완성!

축하합니다. 첫 번째 grim 다이어그램을 완성했습니다. 현재 URL을 복사해 다른 사람과 공유할 수 있습니다.

### 다음에 시도해볼 것

- 노드를 그룹(`{ }`)으로 묶어 중첩 구조 만들기 → [기본 문법](syntax/basics)
- `@shape`으로 노드 모양 바꾸기 → [Shape 종류](reference/shapes)
- 엣지에 웨이포인트 추가하기 → [단축키 & 인터랙션](reference/keyboard)
