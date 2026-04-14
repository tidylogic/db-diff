# db-diff

**데이터베이스 스키마 비교 도구** - MySQL과 PostgreSQL 간의 데이터베이스 스키마 차이를 빠르고 정확하게 비교합니다.

## 개요

`db-diff`는 서로 다른 환경(Dev, QA, Prod)의 데이터베이스 스키마 간 차이를 감지하고 시각화하는 CLI 도구입니다. 개발자, DBA, DevOps 엔지니어가 데이터베이스 일관성을 유지하는 데 도움을 줍니다.

### 주요 특징

- **다중 데이터베이스 지원**: MySQL, PostgreSQL 지원 (확장 가능한 아키텍처)
- **정밀한 비교 엔진**: 테이블, 컬럼, 인덱스, 제약조건, 뷰 등 스키마 전반 비교
- **유연한 출력 형식**: 터미널 친화적 테이블 형식과 JSON 형식
- **선택적 필터링**: 특정 테이블/컬럼 제외 가능
- **마이그레이션 SQL 자동 생성**: 스키마 차이를 DDL로 자동 변환
- **YAML 설정 지원**: 복잡한 비교 환경을 파일로 관리

## 설치

### 사전 요구사항
- Go 1.26 이상

### 빌드

```bash
go build -o db-diff ./cmd/db-diff
```

### Docker (옵션)
```bash
docker build -t db-diff .
docker run --rm db-diff compare --help
```

## 사용법

### 기본 사용

```bash
./db-diff compare \
  --source "mysql://user:pass@localhost:3306/db1" \
  --target "mysql://user:pass@localhost:3307/db2"
```

### 다양한 데이터베이스 비교

```bash
# MySQL과 PostgreSQL 비교 불가능 (같은 방언 필요)
# 같은 DBMS 내에서만 비교 가능
./db-diff compare \
  --source "postgres://user:pass@localhost:5432/db1" \
  --target "postgres://user:pass@localhost:5433/db2"
```

### YAML 설정 파일 사용

프로젝트 루트에 `db-diff.yaml` 파일을 생성하거나 `--config` 옵션으로 지정:

```yaml
source:
  dsn: "mysql://user:pass@dev-db:3306/myapp"
  name: "Dev Database"

target:
  dsn: "mysql://user:pass@prod-db:3306/myapp"
  name: "Prod Database"

output: "table"  # table 또는 json

schema: "myapp"  # DSN의 경로 부분 덮어쓰기 (선택)

ignore:
  tables:
    - "logs"
    - "temp_*"
  fields:
    - "created_at"
    - "updated_at"

migrate:
  enabled: true
  direction: "source_to_target"  # target_to_source도 가능
  output: "migrate.sql"
```

### 마이그레이션 SQL 생성

```bash
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --migrate \
  --migrate-direction source_to_target \
  --migrate-output migration.sql
```

### 필터링 옵션

```bash
# 특정 테이블 제외
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --ignore-tables "logs,sessions,temp_*"

# 특정 컬럼 제외 (모든 테이블에서)
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --ignore-fields "created_at,updated_at"
```

### 출력 형식

#### 테이블 형식 (기본)
```
$ ./db-diff compare --source "mysql://..." --target "mysql://..."

Source: Dev Database
Target: Prod Database

MISSING TABLES (in Target):
- users_temp (18 columns)

DIFFERENT TABLES:
- users
  - Column 'email': VARCHAR(100) -> VARCHAR(255)
  - Column 'is_admin': MISSING (in Target)
  - Index 'idx_email': MISSING (in Target)

- products
  - Column 'price': DECIMAL(10,2) -> DECIMAL(12,3)
  - Constraint 'fk_category': MISSING (in Target)
```

#### JSON 형식
```bash
./db-diff compare \
  --source "mysql://..." \
  --target "mysql://..." \
  --output json | jq .
```

```json
{
  "source": { "name": "Dev Database", "tables_count": 15 },
  "target": { "name": "Prod Database", "tables_count": 14 },
  "differences": {
    "missing_in_target": [
      {
        "name": "users_temp",
        "columns": 18,
        "type": "table"
      }
    ],
    "different": [
      {
        "name": "users",
        "changes": [
          {
            "type": "column_type_change",
            "field": "email",
            "from": "VARCHAR(100)",
            "to": "VARCHAR(255)"
          }
        ]
      }
    ]
  }
}
```

## 아키텍처

### 핵심 모듈

```
internal/
├── connector/        # 데이터베이스 연결 (MySQL, PostgreSQL)
├── schema/          # 스키마 모델 정의
├── diff/            # 비교 엔진
├── output/          # 결과 출력 (테이블, JSON)
├── migrate/         # DDL 생성기
└── config/          # 설정 관리
```

### 처리 흐름

1. **설정 로드**: YAML 파일 또는 CLI 플래그에서 설정 읽음
2. **데이터베이스 연결**: Source와 Target DB에 연결
3. **스키마 추출**: 양쪽 DB에서 스키마 메타데이터 병렬 수집
4. **비교 실행**: 테이블, 컬럼, 인덱스, 제약조건 비교
5. **결과 출력**: 테이블 또는 JSON 형식으로 결과 표시
6. **마이그레이션 생성** (선택): 차이를 DDL로 변환하여 파일 저장

## 지원되는 비교 항목

| 항목 | MySQL | PostgreSQL | 비고 |
|------|-------|-----------|------|
| 테이블 존재 여부 | ✓ | ✓ | Missing/Extra |
| 컬럼 정의 | ✓ | ✓ | 타입, NULL, 기본값 |
| 데이터 타입 | ✓ | ✓ | 정밀 비교 |
| PRIMARY KEY | ✓ | ✓ | 컬럼 순서 포함 |
| UNIQUE INDEX | ✓ | ✓ | |
| FOREIGN KEY | ✓ | ✓ | 제약조건 명 |
| 일반 INDEX | ✓ | ✓ | |
| 컬럼 주석/설명 | ✓ | ✓ | DB 메타데이터 |
| 뷰 | 계획 중 | 계획 중 | |
| 트리거 | 계획 중 | 계획 중 | |

## 제한사항

- **같은 DBMS만 비교 가능**: MySQL ↔ MySQL 또는 PostgreSQL ↔ PostgreSQL
- **읽기 전용**: 비교만 수행하며 자동 동기화는 미지원
- **프로시저/트리거**: 현재 비교 대상 미포함 (계획 중)

## CLI 옵션

```bash
./db-diff compare --help

Usage:
  db-diff compare [flags]

Flags:
  --config string           YAML 설정 파일 경로 (기본: auto-discover db-diff.yaml)
  --source string           Source DSN (예: "mysql://user:pass@host:3306/db")
  --source-name string      Source 표시 이름 (예: "DEV")
  --target string           Target DSN
  --target-name string      Target 표시 이름 (예: "QA")
  --output string           출력 형식: "table" 또는 "json" (기본: table)
  --schema string           스키마 이름 (DSN의 경로 부분 덮어쓰기)
  --ignore-tables string    제외할 테이블 목록 (쉼표 구분)
  --ignore-fields string    제외할 컬럼 목록 (쉼표 구분)
  --migrate                 마이그레이션 SQL 생성 활성화
  --migrate-direction string "source_to_target" 또는 "target_to_source" (기본: source_to_target)
  --migrate-output string   마이그레이션 파일 경로 (기본: migrate.sql)
  -h, --help                도움말 표시
```

## 예제

### 1. 개발/프로덕션 DB 비교

```bash
./db-diff compare \
  --source "mysql://dev_user:dev_pass@dev-db.example.com:3306/myapp" \
  --source-name "Development" \
  --target "mysql://prod_user:prod_pass@prod-db.example.com:3306/myapp" \
  --target-name "Production"
```

### 2. QA 환경과 템플릿 DB 비교

```bash
./db-diff compare \
  --config deploy/qa-check.yaml \
  --output json > qa-report.json
```

### 3. 마이그레이션 스크립트 자동 생성

```bash
./db-diff compare \
  --source "mysql://staging:pass@staging-db:3306/shop" \
  --target "mysql://staging:pass@staging-db-new:3306/shop" \
  --migrate \
  --migrate-output scripts/migration-$(date +%Y%m%d).sql
```

## 기여하기

버그 리포트, 기능 제안, Pull Request는 환영합니다!

### 개발 환경 설정

```bash
# 저장소 클론
git clone https://github.com/tidylogic/db-diff.git
cd db-diff

# 테스트 실행
go test ./...

# 빌드
go build -o db-diff ./cmd/db-diff
```

### 테스트

Testcontainers를 활용하여 실제 MySQL/PostgreSQL 환경에서 통합 테스트 수행:

```bash
go test -v ./internal/connector/...
```

## 라이센스

MIT License - 자세한 내용은 [LICENSE](LICENSE) 참조

## 로드맵

### Phase 1 (Core) ✓
- 기본 아키텍처 및 MySQL/PostgreSQL 지원
- 정밀 비교 엔진
- JSON 출력 및 마이그레이션 생성

### Phase 2 (Advanced)
- YAML 설정 완성
- 추가 DBMS 지원 (Oracle, SQL Server)
- 성능 최적화

### Phase 3 (GUI) 계획 중
- 웹 기반 시각화 도구
- DDL 생성기 고도화
- 실시간 동기화 기능

### Phase 4 (안정성) 계획 중
- 통합 테스트 자동화
- 다양한 DBMS 버전 호환성 검증
- 사용자 문서화 및 튜토리얼

## 문제 해결

### "cannot compare MySQL and PostgreSQL"
- Source와 Target이 같은 DBMS여야 합니다.
- DSN 스키마 확인: `mysql://` 또는 `postgres://`

### 연결 거부 에러
```bash
# 1. 데이터베이스 접근성 확인
mysql -h <host> -u <user> -p<password>

# 2. DSN 형식 확인
# 올바른 형식: mysql://user:password@host:port/database
# 주의: 암호에 @ 기호 포함 시 URL 인코딩 필요 (예: %40)
```

### 권한 부족 에러
- 데이터베이스 사용자에게 다음 권한 필요:
  - MySQL: `SELECT` (information_schema)
  - PostgreSQL: `CONNECT`, `USAGE` (스키마)

## 지원

- 📧 버그 리포트: [GitHub Issues](https://github.com/tidylogic/db-diff/issues)
- 📝 문서: 프로젝트 Wiki 참조
- 💬 토론: GitHub Discussions

## 변경 이력

### v0.1.0 (초기 릴리스)
- MySQL 및 PostgreSQL 기본 지원
- 스키마 비교 및 마이그레이션 생성
- YAML 설정 파일 지원
