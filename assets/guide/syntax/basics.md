# 기본 문법

## 노드 선언

노드는 ID만으로 선언합니다.

```grim
my-node
```

라벨을 붙이려면 콜론 뒤에 문자열을 씁니다.

```grim
my-node: "표시될 텍스트"
```

### 노드 ID 규칙

- 영문 대소문자로 시작 (`[a-zA-Z]`)
- 이후 영문, 숫자, 하이픈(`-`), 언더스코어(`_`) 허용
- 점(`.`)은 경로 구분자 전용이므로 ID에 사용 불가
- 한글, 공백, 특수문자는 미지원

```grim
# 유효한 ID
my-node
server1
API-Gateway
my_service

# 유효하지 않은 ID
# 1node      # 숫자로 시작
# my node    # 공백 포함
# my.node    # 점은 경로 구분자
```

---

## 엣지 (연결선)

두 노드를 연결합니다.

```grim
# 단방향 (오른쪽으로)
A -> B

# 단방향 (왼쪽으로)
A <- B

# 양방향
A <-> B

# 화살표 없는 선
A -- B
```

### 엣지 라벨

엣지에도 라벨을 붙일 수 있습니다.

```grim
client -> server: "HTTP 요청"
server <- database: "쿼리 결과"
```

---

## 노드 중첩

`{ }` 블록으로 노드를 중첩할 수 있습니다. 부모 노드 안에 자식 노드가 그려집니다.

```grim
network {
    frontend
    backend
    database
}
```

그룹 내부에서도 엣지를 정의할 수 있습니다.

```grim
network {
    frontend -> backend
    backend -> database
}
```

---

## 경로 참조

점(`.`)으로 중첩된 노드를 참조합니다.

```grim
service {
    api
    worker
}

# 그룹 외부에서 내부 노드를 참조
service.api -> service.worker
external -> service.api
```

---

## 주석

`#` 뒤의 내용은 주석으로 처리됩니다.

```grim
# 이것은 주석입니다
client -> server  # 인라인 주석도 가능합니다
```

---

## 암시적 노드 선언

엣지에 등장하는 노드는 따로 선언하지 않아도 자동으로 생성됩니다.

```grim
# A, B, C를 별도로 선언하지 않아도 됩니다
A -> B -> C
```

마우스로 이동하면 자동으로 명시적 선언으로 변환됩니다.

---

## 다음 단계

- [메타데이터 지시자](metadata) — `@layout`, `@style` 등의 상세 문법을 배웁니다.
- [Shape 종류](../reference/shapes) — 10종의 노드 모양을 확인합니다.
