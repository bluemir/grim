# 스타일 속성

`@style` 지시자로 노드와 엣지의 시각적 속성을 지정합니다.

---

## 노드 스타일 속성

```grim
@style {fill: blue-500, stroke: black, stroke-width: 2, font-color: white, shadow: 3}
my-node
```

| 속성 | 설명 | 예시 값 |
|------|------|---------|
| `fill` | 배경 색상 | `blue-500`, `#3498db`, `transparent` |
| `stroke` | 테두리 색상 | `black`, `#333` |
| `stroke-width` | 테두리 두께 (px) | `2` |
| `stroke-dash` | 테두리 점선 패턴 | `5,3` |
| `shadow` | 그림자 크기 (px, 0이면 없음) | `3` |
| `opacity` | 투명도 (0~1) | `0.5` |
| `font-size` | 텍스트 크기 (px) | `14` |
| `font-color` | 텍스트 색상 | `white`, `#fff` |
| `border-radius` | 모서리 둥글기 (px) | `8` |

---

## 엣지 스타일 속성

```grim
A -> B {
    @style {stroke: red, stroke-width: 2, stroke-dash: 5,3}
}
```

| 속성 | 설명 | 예시 값 |
|------|------|---------|
| `stroke` | 선 색상 | `red`, `#333` |
| `stroke-width` | 선 두께 (px) | `2` |
| `stroke-dash` | 점선 패턴 | `5,3` |
| `arrow` | 화살표 모양 | `triangle`, `diamond`, `none` |

---

## 색상 표기법

### CSS 형식

표준 CSS 색상 표기를 지원합니다.

```grim
@style {fill: #3498db}
@style {fill: rgb(52, 152, 219)}
@style {fill: rgba(52, 152, 219, 0.5)}
```

### 편의 색상

```grim
@style {fill: black}
@style {fill: white}
@style {fill: transparent}
```

### Material Design 색상 팔레트

`{색상 이름}-{단계}` 형식으로 사용합니다.

```grim
@style {fill: blue-500}
@style {fill: gray-100}
@style {fill: red-900}
```

**단계**: `50`, `100`, `200`, `300`, `400`, `500`, `600`, `700`, `800`, `900`

**색상 계열 (19가지)**:

| 계열 | 예시 |
|------|------|
| `red` | `red-500` |
| `pink` | `pink-300` |
| `purple` | `purple-700` |
| `deep-purple` | `deep-purple-500` |
| `indigo` | `indigo-500` |
| `blue` | `blue-500` |
| `light-blue` | `light-blue-300` |
| `cyan` | `cyan-500` |
| `teal` | `teal-500` |
| `green` | `green-500` |
| `light-green` | `light-green-300` |
| `lime` | `lime-500` |
| `yellow` | `yellow-500` |
| `amber` | `amber-500` |
| `orange` | `orange-500` |
| `deep-orange` | `deep-orange-500` |
| `brown` | `brown-500` |
| `gray` | `gray-500` |
| `blue-gray` | `blue-gray-500` |

일부 색상은 `alt` 변형을 지원합니다. (`blue-alt-200`, `blue-alt-700` 등)

---

## 스타일 예시

```grim
# 강조 노드
@style {fill: blue-500, font-color: white, shadow: 4, border-radius: 8}
primary-node: "강조 표시"

# 비활성 노드
@style {fill: gray-200, stroke: gray-400, opacity: 0.6}
disabled-node: "비활성"

# 점선 테두리
@style {stroke: red-500, stroke-dash: 5,3, fill: transparent}
boundary: "경계선"

# 점선 엣지
A -> B {
    @style {stroke: blue-300, stroke-dash: 4,4, stroke-width: 2}
}
```
