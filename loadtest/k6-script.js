import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  vus: 20,
  duration: '2m',
};

const baseUrl = __ENV.BASE_URL || 'http://localhost:8080';

export default function () {
  const createRes = http.post(`${baseUrl}/todos`, JSON.stringify({ title: `task-${__VU}-${__ITER}` }), {
    headers: { 'Content-Type': 'application/json' },
  });

  check(createRes, {
    'create status is 201': (r) => r.status === 201,
  });

  const listRes = http.get(`${baseUrl}/todos`);
  check(listRes, {
    'list status is 200': (r) => r.status === 200,
  });

  sleep(0.2);
}
