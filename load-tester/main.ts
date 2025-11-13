import http from 'k6/http'
import { htmlReport } from 'https://raw.githubusercontent.com/benc-uk/k6-reporter/latest/dist/bundle.js'

export const options = {
    // iterations: 50,

    // thresholds: {
    //     http_req_duration: ['p(95)<1000'],
    // },
    //
    // stages: [
    //     { duration: '40s', target: 100 },
    //     { duration: '80s', target: 200 },
    //     { duration: '1m', target: 250 },
    //     { duration: '1.5m', target: 0 },
    // ],
    //
    summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99.99)', 'count'],

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

const file = open("/home/mazksr/Walls/beach.png", "b")
const ALGORITHM = __ENV.ALG || ""

export default function () {
    const data = {
        file: http.file(file)
    };

    http.post("http://localhost:8080/"+ALGORITHM, data)
}

export function handleSummary(data) {
    return {
        'summary.html': htmlReport(data),
        'summary.json': JSON.stringify(data),
    }
}
