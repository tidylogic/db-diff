# [Project Plan] Cross-Platform RDBMS Schema Diff Tool
본 프로젝트는 서로 다른 환경(Dev, QA, Prod) 간의 데이터베이스 스키마 오차를 최소화하고, 협업 및 운영 효율을 높이기 위한 범용 데이터베이스 스키마 비교 도구 개발을 목표로 합니다.

## 기술 스택
- CLI Tool
  - Golang 1.26
  - Docker
  - Testcontainers
  - golang SQL Drivers(https://go.dev/wiki/SQLDrivers)
- Web GUI
  - HTML, CSS, Typescript(React)
  - Shadcn UI

## 디렉토리 구조

1. 프로젝트 개요
   - 목적: 이종/동종 RDBMS 간의 스키마 및 메타데이터 비교 분석
   - 주요 타겟: 개발자, DBA, DevOps 엔지니어 
   - 핵심 가치: 자동화 용이성(JSON Output), 시각적 직관성(Web GUI), 확장성(Modular Architecture)
2. 핵심 기능 (CLI Tool)
   - 2.1. 다중 데이터베이스 지원 (Multi-Dialect)
     - 지원 대상: MySQL, PostgreSQL, Oracle, SQL Server 등 주요 RDBMS 지원.
     - 확장 구조: Interface 기반 설계로 새로운 DBMS 커넥터 추가가 용이한 모듈화 구조.

   - 2.2. 정밀 비교 엔진 (Deep Diff)
     - 비교 범위: 테이블(Table), 컬럼(Column), 인덱스(Index), 제약 조건(Constraints), 뷰(View) 및 구체화된 뷰(Materialized View) 등 스키마 전반.
     - 상세 속성: 데이터 타입, Null 허용 여부, 기본값, 코멘트, 길이 등 메타데이터 상세 비교.
     - 선택적 비교: 특정 항목(예: 인덱스 제외, 특정 테이블만 포함 등)을 사용자가 필터링할 수 있는 옵션 제공.

   - 2.3 인터페이스 및 설정
     - 입력: 명령어 라인 인자(Flags) 및 YAML 설정 파일을 통한 복잡한 비교 환경 구성 지원.
     - 출력: 
       - JSON Format: 타 프로그램(CI/CD 파이프라인 등)과의 연동을 위한 표준 데이터 포맷 제공. 
       - Human-Friendly Terminal UI: 터미널 내에서 가독성 높은 비교 결과 요약 출력.

3. 웹 기반 GUI 확장 계획
   - 3.1 시각화 및 사용자 경험 (Web-based)
     - 비주얼 디프(Visual Diff): 테이블 구조 및 변경 사항을 나란히(Side-by-side) 배치하여 시각적으로 강조.
     - 대시보드: 전체적인 스키마 불일치 현황을 한눈에 파악할 수 있는 요약 정보 제공.

   - 3.2 관리 및 수정 기능
     - DDL 생성 엔진: 비교 결과의 차이점을 해소하기 위한 CREATE, ALTER, DROP 등 동적 DDL SQL 자동 생성.
     - 직접 수정(Sync): GUI 상에서 차이점을 확인한 후, 타겟 DB에 즉시 반영하거나 스크립트를 추출할 수 있는 기능.

4. 기술적 요구사항
   - 4.1 성능 최적화
     - 대규모 스키마 처리: 대규모 메타데이터 조회 시 성능 저하를 방지하기 위한 최적화된 쿼리 알고리즘 적용.
     - 병렬 처리: 데이터베이스별 메타데이터 추출 및 비교 프로세스를 병렬화하여 수행 시간 단축.

   - 4.2 안정성 및 확장성
     - 모듈화 설계: 비교 로직과 데이터 추출 로직을 분리하여 유지보수성 확보.
     - 테스트 자동화: 다양한 DBMS 버전 및 케이스별 시나리오를 포함한 테스트 슈트 구축으로 신뢰성 확보.
       - 이때는 testcontainers 라이브러리를 활용하여 실제 DB 환경에서의 통합 테스트를 자동화하는 방안을 고려.
       - 동일한 DB 여도 메이저한 버전들 여러개를 테스트하여 호환성 검증.

5. 단계별 개발 로드맵
   - Phase 1 (Core): 기본 아키텍처 설계, MySQL/PostgreSQL 커넥터 개발, 핵심 비교 엔진 및 JSON 출력 구현.
   - Phase 2 (Advanced): YAML 설정 지원, 추가 DBMS(Oracle, SQL Server) 확장, 성능 최적화.
   - Phase 3 (GUI): 웹 기반 시각화 툴 개발, DDL 생성기 및 동기화 기능 통합.
   - Phase 4 (Stability): 통합 테스트 자동화 및 사용자 문서화.