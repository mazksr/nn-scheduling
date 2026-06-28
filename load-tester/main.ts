import http from 'k6/http'
import { Counter } from 'k6/metrics'
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/latest/dist/bundle.js'

export const options = {
    thresholds: {
        http_req_failed: ['rate<0.05'], // Less than 5% failure rate
    },

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.99)', 'count'],

    scenarios: {
        contacts: {
            executor: 'ramping-arrival-rate',

            // Start iterations per `timeUnit`
            startRate: 20,

            // Start `startRate` iterations per minute
            timeUnit: '1s',

            // Pre-allocate necessary VUs.
            preAllocatedVUs: 100,
            maxVUs: 600,

            stages: [
                { target: 50, duration: '30s' },

                { target: 200, duration: '30s' },
                { target: 200, duration: '60s' },

                { target: 500, duration: '30s' },
                { target: 500, duration: '90s' },

                { target: 30, duration: '1m' },
            ],
        },
    },
}

const workerCounters: { [key: string]: Counter } = {
    'worker_1': new Counter('worker_1_requests'),
    'worker_2': new Counter('worker_2_requests'),
    'worker_3': new Counter('worker_3_requests'),
    'worker_4': new Counter('worker_4_requests'),
    'worker_5': new Counter('worker_5_requests'),
    'worker_6': new Counter('worker_6_requests'),
    'worker_7': new Counter('worker_7_requests'),
    'worker_8': new Counter('worker_8_requests'),
    'unknown': new Counter('worker_unknown_requests'),
}

const imageListText = open('./images.txt');
const imagePaths = imageListText.split('\n').map(s => s.trim()).filter(s => s.length > 0);

const imageFiles = imagePaths.map(filepath => {
    return {
        name: filepath.split('/').pop(),
        data: open(filepath, 'b')
    };
});

const ALGORITHM = __ENV.ALG || ""

export default function () {
    const randomImg = imageFiles[Math.floor(Math.random() * imageFiles.length)];
    
    const data = {
        file: http.file(randomImg.data, randomImg.name)
    };

    const res = http.post("http://localhost:8080/"+ALGORITHM, data)

    const workerId = res.headers['X-Worker-Id'] || res.headers['x-worker-id'] || 'unknown';
    const counter = workerCounters[workerId] || workerCounters['unknown'];
    counter.add(1);
}

export function handleSummary(data) {
    return {
        'summary.html': htmlReport(data),
        'summary.json': JSON.stringify(data),
    }
}
