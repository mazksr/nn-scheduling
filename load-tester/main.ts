import http from 'k6/http'
import { Counter } from 'k6/metrics'
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/latest/dist/bundle.js'
import { SharedArray } from 'k6/data'

export const options = {
    thresholds: {
        http_req_failed: ['rate<0.05'], // Less than 5% failure rate
    },

    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99)', 'p(99.99)', 'count'],

    scenarios: {
            contacts: {
                executor: 'ramping-arrival-rate',
                startRate: 2, // Start gently to let your Go proxy boot up
                timeUnit: '1s',

                // Little's Law Calculation:
                // Peak RPS is 30. If under heavy load, an image takes max 4 seconds.
                // 30 RPS * 4 seconds = 120 VUs needed to hold connections open.
                preAllocatedVUs: 20,
                maxVUs: 150, // Safe ceiling that won't eat all your testing machine's RAM

                stages: [
                    // Stage 1: Warmup & Calibration (30 seconds)
                    // Allows your proxy's Exponential Moving Average (EMA) to adjust
                    // the initial bootstrapped alpha to the actual hardware reality.
                    { target: 10, duration: '30s' },

                    // Stage 2: Sustained Load (1 minute)
                    // Tests the baseline stability of the system.
                    { target: 24, duration: '1m' },

                    // Stage 3: The "Thundering Herd" Burst (30 seconds)
                    // THIS IS YOUR THESIS MONEY-MAKER.
                    // Pushes the system past its comfort zone. Round Robin and
                    // Least Connections will choke here. Payload-Aware should survive.
                    { target: 60, duration: '30s' },

                    // Stage 4: Sustained High Load (30 seconds)
                    // Keeps the pressure on to see if the CPU bottlenecks compound.
                    { target: 60, duration: '30s' },

                    // Stage 5: Cooldown / Recovery (30 seconds)
                    // Drops traffic back down to see how fast the W_i queues drain
                    // and return to normal latency.
                    { target: 10, duration: '30s' },
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

// ✅ GOOD: Loads into memory once and shares across all VUs
const imageFiles = new SharedArray('image files', function () {
    const imageListText = open('./images.txt');
    const imagePaths = imageListText.split('\n').map(s => s.trim()).filter(s => s.length > 0);

    return imagePaths.map(filepath => {
        return {
            name: filepath.split('/').pop(),
            data: open(filepath, 'b') // open() must be called inside this function
        };
    });
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
