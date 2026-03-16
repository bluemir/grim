# Shape 종류

`@shape` 지시자로 노드의 모양을 지정합니다. 기본값은 `rectangle`입니다.

## Shape 목록

| Shape | 이름 | 설명 | 코드 예시 |
|-------|------|------|-----------|
| □ | `rectangle` | 일반 사각형 (기본값) | `@shape rectangle` |
| ▢ | `rounded` | 모서리가 둥근 사각형 | `@shape rounded` |
| ○ | `circle` | 원 | `@shape circle` |
| ◇ | `diamond` | 다이아몬드 (의사결정) | `@shape diamond` |
| ⊓ | `cylinder` | 원통 (데이터베이스) | `@shape cylinder` |
| ☁ | `cloud` | 구름 | `@shape cloud` |
| ⬡ | `hexagon` | 육각형 | `@shape hexagon` |
| ▱ | `parallelogram` | 평행사변형 | `@shape parallelogram` |
| 👤 | `user` | 사람 형태 (액터) | `@shape user` |
| 🤖 | `bot` | 봇/AI 형태 | `@shape bot` |

## 코드 예시

```grim
@shape rectangle
node-a: "일반 사각형"

@shape rounded
node-b: "둥근 사각형"

@shape circle
node-c: "원"

@shape diamond
node-d: "다이아몬드"

@shape cylinder
database: "데이터베이스"

@shape cloud
cloud-service: "클라우드"

@shape hexagon
node-e: "육각형"

@shape parallelogram
node-f: "평행사변형"

@shape user
actor: "사용자"

@shape bot
ai: "AI 에이전트"
```

## 사용 팁

- `cylinder`는 데이터베이스를 표현할 때 주로 사용합니다.
- `diamond`는 플로우차트에서 조건 분기를 표현할 때 사용합니다.
- `user`와 `bot`은 시퀀스 다이어그램이나 시스템 컨텍스트 다이어그램에 유용합니다.
- `@style {border-radius: 8}`로 `rectangle`에 부분적인 둥근 모서리를 줄 수도 있습니다.
