# db-diff

[![CI](https://github.com/tidylogic/db-diff/actions/workflows/ci.yml/badge.svg)](https://github.com/tidylogic/db-diff/actions/workflows/ci.yml)

**데이터베이스 스키마 비교 도구** - MySQL과 PostgreSQL 간의 데이터베이스 스키마 차이를 빠르고 정확하게 비교합니다.

🇬🇧 [English Documentation](README.md)

## 개요

`db-diff`는 서로 다른 환경(Dev, QA, Prod)의 데이터베이스 스키마 간 차이를 감지하고 시각화하는 CLI 도구입니다. 개발자, DBA, DevOps 엔지니어가 데이터베이스 일관성을 유지하는 데 도움을 줍니다.

### 주요 특징

- **다중 데이터베이스 지원**: MySQL, PostgreSQL 지원 (확장 가능한 아키텍처)
- **정밀한 비교 엔진**: 테이블, 컬럼, 인덱스, 제약조건, 뷰 등 스키마 전반 비교
- **유연한 출력 형식**: 터미널 친화적 테이블 형식과 JSON 형식
- **선택적 필터링**: 특정 테이블/컬럼 제외 가능
- **마이그레이션 SQL 자동 생성**: 스키마 차이를 DDL로 자동 변환
- **YAML 설정 지원**: 복잡한 비교 환경을 파일로 관리
- **Web GUI**: 브라우저 기반 diff 뷰어 및 인터랙티브 마이그레이션 SQL 빌더

## 설치

### 사전 요구사항
- Go 1.26 이상
- Node.js 18+ 및 npm (웹 GUI 빌드 시에만 필요)

### 방법 1: `go install` 사용 (권장)

```bash
go install github.com/tidylogic/db-diff/cmd/db-diff@latest
```

바이너리가 `$GOPATH/bin/db-diff` (일반적으로 `$HOME/go/bin/db-diff`)에 설치됩니다.

`$GOPATH/bin`이 PATH에 포함되어 있는지 확인하세요:
```bash
# ~/.bashrc, ~/.zshrc, 또는 셸 설정 파일에 추가
export PATH=$PATH:$HOME/go/bin
```

### 방법 2: 소스에서 빌드

```bash
git clone https://github.com/tidylogic/db-diff.git
cd db-diff

# 프론트엔드 + 백엔드 전체 빌드
make all

# Go 바이너리만 빌드 (web/static/ 이 이미 있는 경우)
go build -o db-diff ./cmd/db-diff
```

바이너리가 현재 디렉토리에 생성됩니다.

### Docker (옵션)
```bash
docker build -t db-diff .
docker run --rm db-diff compare --help
```

## Web GUI

브라우저 기반 diff 뷰어로 스키마 변경사항을 시각적으로 확인하고 마이그레이션 SQL을 인터랙티브하게 생성합니다:

```bash
# 1. JSON diff 파일 생성
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --output json > diff.json

# 2. 웹 서버 실행 (기본 포트 8080)
./db-diff web

# 3. http://localhost:8080 접속 후 diff.json 파일 로드
```

```bash
# 포트 변경
./db-diff web --port 3000
```

### Web UI 빌드

```bash
# 프론트엔드 + 백엔드 한 번에 빌드
make all

# 개별 빌드
make ui       # npm install + vite build → web/static/
make build    # go build

# 프론트엔드 개발 서버 (http://localhost:5173, 핫 리로드)
make dev-ui
```

### Web GUI 기능

| 기능 | 설명 |
|------|------|
| **통계 바** | Source Only / Target Only / Modified 건수 한눈에 확인 |
| **테이블/뷰 목록** | 접기/펼치기 섹션, 변경 유형 필터 칩, DB 이름 토글이 있는 사이드바; **All** 버튼은 현재 필터에 보이는 항목만 선택 |
| **상세 뷰** | 컬럼/인덱스/제약조건별 before→after 값 비교 |
| **마이그레이션 빌더** | 방향(src→tgt / tgt→src)과 방언(MySQL/PostgreSQL) 토글 |
| **선택적 SQL 생성** | 항목별 체크/언체크로 포함할 변경사항 선택 |
| **복사 / 다운로드** | SQL을 클립보드에 복사하거나 `.sql` 파일로 다운로드 |
| **테마** | Light / Dark / System (OS 환경설정 연동, localStorage 저장) |

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
  direction: "apply_to_target"  # apply_to_source도 가능
  output: "migrate.sql"
```

### 마이그레이션 SQL 생성

```bash
./db-diff compare \
  --source "mysql://user:pass@host1:3306/db" \
  --target "mysql://user:pass@host2:3306/db" \
  --migrate \
  --migrate-direction apply_to_target \
  --migrate-output migration.sql
```

**마이그레이션 방향 의미:**

| 방향 | 의미 | SQL 적용 대상 |
|------|------|--------------|
| `apply_to_target` | source 스키마 → target으로 전파 (target을 source에 맞춤) | TARGET 데이터베이스 |
| `apply_to_source` | target 스키마 → source로 전파 (source를 target에 맞춤) | SOURCE 데이터베이스 |

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
  --migrate-direction string "apply_to_target" 또는 "apply_to_source" (기본: apply_to_target)
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

Testcontainers를 활용하여 실제 데이터베이스 컨테이너를 실행하고 여러 메이저 버전에서 스키마 추출을 검증합니다:

| 데이터베이스 | 테스트 버전           |
|--------------|-----------------------|
| MySQL        | 5.7, 8.0, 8.4         |
| PostgreSQL   | 13, 14, 15, 16, 17    |

모든 버전 서브테스트는 병렬로 실행됩니다. 호스트에 Docker가 설치되어 있어야 합니다.

컨테이너 기반 테스트는 `integration` 빌드 태그를 사용하며, 기본 `go test ./...` 실행에서는 제외됩니다. 실행하려면 `-tags integration`을 전달해야 합니다:

```bash
# 전체 통합 테스트 실행 (Docker 필요)
go test -v -timeout 20m -tags integration ./...

# 커넥터 호환성 테스트만 실행
go test -v -timeout 15m -tags integration ./internal/connector/...

# 마이그레이션 통합 테스트만 실행
go test -v -timeout 15m -tags integration ./internal/migrate/...
```

> **CI / GitHub Actions**: 기본 CI 실행(`go test ./...`)은 `integration` 빌드 태그를 통해 컨테이너 테스트를 제외하므로 Docker 없이도 CI가 빠르게 통과됩니다. Docker가 설치된 환경에서 `go test -tags integration`으로 로컬 실행하세요.

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

### Phase 3 (GUI) ✓
- 웹 기반 diff 뷰어 (React + TypeScript + Tailwind CSS)
- 항목별 선택 기능이 있는 인터랙티브 마이그레이션 SQL 빌더
- Dark / Light / System 테마 토글

### Phase 4 (안정성) ✓
- 비교 엔진(`internal/diff`) 및 마이그레이션 SQL 생성기(`internal/migrate`) 단위 테스트 추가
- testcontainers를 이용한 MySQL 5.7/8.0/8.4 및 PostgreSQL 13–17 실제 DB 통합 테스트
- Go `POST /api/migrate` 엔드포인트 추가 — TypeScript SQL 생성 코드 제거, Go로 DDL 생성 일원화
- `execSQL` 테스트 헬퍼 버그 수정 — 주석 헤더 다음에 오는 SQL 구문이 올바르게 실행되도록 수정

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
- 접속 사용자가 특정 뷰에 대한 `SELECT` 권한이 없으면 PostgreSQL은 `information_schema.views`의 `view_definition`을 NULL로 반환합니다. 이 경우 뷰는 빈 정의로 기록되며 에러 없이 처리됩니다.

## 지원

- 📧 버그 리포트: [GitHub Issues](https://github.com/tidylogic/db-diff/issues)
- 📝 문서: 프로젝트 Wiki 참조
- 💬 토론: GitHub Discussions

## 변경 이력

### 미출시
- 수정: 뷰에 SELECT 권한이 없을 때 PostgreSQL `view_definition` NULL 오류 수정
- 수정: Columns/Indexes/Constraints가 null인 JSON 로드 시 웹 UI 크래시 수정
- 수정: 추가/삭제된 테이블의 마이그레이션 SQL이 주석 처리된 플레이스홀더 대신 올바른 `CREATE TABLE` 구문을 생성합니다. `apply_to_source` 방향으로 target-only 테이블 선택 시 마이그레이션 패널이 빈 화면으로 표시되던 버그도 함께 수정됩니다.

### v0.1.0 (초기 릴리스)
- MySQL 및 PostgreSQL 기본 지원
- 스키마 비교 및 마이그레이션 생성
- YAML 설정 파일 지원
