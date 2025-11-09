import http from 'k6/http'

export const options = {
    // iterations: 50,

    // summaryTrendStats: ['avg', 'min', 'med', 'max', 'p(90)', 'p(95)', 'p(99.99)', 'count'],
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

    scenarios: {
        contacts: {
            executor: 'ramping-arrival-rate',

            // Start iterations per `timeUnit`
            startRate: 20,

            // Start `startRate` iterations per minute
            timeUnit: '1s',

            // Pre-allocate necessary VUs.
            preAllocatedVUs: 100,
            maxVUs: 500,

            stages: [
                { target: 20, duration: '30s' },

                { target: 50, duration: '30s' },
                { target: 50, duration: '60s' },

                { target: 100, duration: '30s' },
                { target: 100, duration: '90s' },

                { target: 10, duration: '1m' },
            ],
        },
    },
}

const file = open("/home/mazksr/Downloads/wifi.webp", "b")
const ALGORITHM = __ENV.ALG || ""

export default function () {
    const data = {
        file: http.file(file)
    };

    http.post("http://localhost:8080/"+ALGORITHM, data)
}

// export function handleSummary(data) {
//   return {
//     'summary.json': JSON.stringify(data),
//   }
// }
