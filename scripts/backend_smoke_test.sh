#!/bin/bash
set -e
BASE=http://localhost:8080/api/v1

echo "=== register candidate ==="
CAND_JSON=$(curl -s -X POST $BASE/auth/register/candidate -H "Content-Type: application/json" \
  -d '{"email":"john.doe@example.com","password":"SecurePass123","full_name":"John Doe","phone":"555-1234"}')
echo "$CAND_JSON" | python3 -m json.tool
CAND_TOKEN=$(echo "$CAND_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['access_token'])")

echo "=== register company ==="
COMP_JSON=$(curl -s -X POST $BASE/auth/register/company -H "Content-Type: application/json" \
  -d '{"email":"hr@exampletech.com","password":"SecurePass123","company_name":"Example Technologies","website":"https://exampletech.com"}')
echo "$COMP_JSON" | python3 -m json.tool
COMPANY_TOKEN=$(echo "$COMP_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['access_token'])")

echo "=== create job ==="
JOB_JSON=$(curl -s -X POST $BASE/jobs -H "Authorization: Bearer $COMPANY_TOKEN" -H "Content-Type: application/json" -d '{
  "title": "Junior Go Backend Developer",
  "description": "We are looking for a junior backend developer to join our platform team, working on REST APIs and services in Go.",
  "location": "Remote", "employment_type": "full_time", "experience_min_years": 1,
  "education_requirement": "Bachelor degree in Computer Science",
  "interview_question_count": 5,
  "skills": [
    {"name": "Go", "is_required": true, "weight": 2},
    {"name": "MySQL", "is_required": true, "weight": 1.5},
    {"name": "REST API", "is_required": true, "weight": 1.5},
    {"name": "Docker", "is_required": false, "weight": 1},
    {"name": "Git", "is_required": false, "weight": 1}
  ]
}')
JOB_ID=$(echo "$JOB_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")
echo "job id: $JOB_ID"

echo "=== publish job ==="
curl -s -X POST $BASE/jobs/$JOB_ID/publish -H "Authorization: Bearer $COMPANY_TOKEN" | python3 -c "import json,sys; d=json.load(sys.stdin)['data']; print('status:', d['status'], '| published_at:', d['published_at'])"

echo "=== upload + parse resume ==="
RESUME_JSON=$(curl -s -X POST $BASE/candidates/me/resume -H "Authorization: Bearer $CAND_TOKEN" -F "resume=@/tmp/john_doe_resume.txt")
echo "$RESUME_JSON" | python3 -m json.tool

echo "=== apply to job ==="
APP_JSON=$(curl -s -X POST $BASE/jobs/$JOB_ID/apply -H "Authorization: Bearer $CAND_TOKEN")
echo "$APP_JSON" | python3 -m json.tool
APP_ID=$(echo "$APP_JSON" | python3 -c "import json,sys; print(json.load(sys.stdin)['data']['id'])")
echo "application id: $APP_ID"

echo "=== get match score ==="
curl -s $BASE/applications/$APP_ID/match -H "Authorization: Bearer $CAND_TOKEN" | python3 -m json.tool

echo "=== company views ranked candidates ==="
curl -s $BASE/jobs/$JOB_ID/matches -H "Authorization: Bearer $COMPANY_TOKEN" | python3 -m json.tool

echo "=== IDOR check: candidate tries to view job applications list (should be 403) ==="
curl -s -o /dev/null -w "%{http_code}\n" $BASE/jobs/$JOB_ID/applications -H "Authorization: Bearer $CAND_TOKEN"

echo "=== company shortlists candidate ==="
curl -s -X PUT $BASE/applications/$APP_ID/status -H "Authorization: Bearer $COMPANY_TOKEN" -H "Content-Type: application/json" -d '{"status":"shortlisted"}' | python3 -m json.tool
