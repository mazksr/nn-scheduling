import http from 'k6/http'

export const options = {
    // iterations: 10,
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99.99)', 'count'],
    stages: [
        { duration: '40s', target: 100 },
        { duration: '80s', target: 200 },
        { duration: '1m', target: 250 },
        { duration: '1.5m', target: 0 },
    ],
}

const file = open("/home/mazksr/Walls/beach.png", "b")

export default function () {
    const data = {
        file: http.file(file)
    };

    http.post("http://localhost:8080/infer", data)
}

export function handleSummary(data) {
  return {
    'summary.json': JSON.stringify(data),
  }
}
