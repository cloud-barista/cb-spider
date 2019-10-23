RESTSERVER=localhost

curl -X POST http://$RESTSERVER:1024/sshrun -H 'Content-Type: application/json' -d '{
    "PrivateKey": [
        "-----BEGIN RSA PRIVATE KEY-----",
        "MIIEoQIBAAKCAQEArVNOLwMIp5VmZ4VPZotcoCHdEzimKalAsz+ccLfvAA1Y2ELH",
        "VwihRvkrqukUlkC7B3ASSCtgxIt5ZqfAKy9JvlT+Po/XHfaIpu9KM/XsZSdsF2jS",
        "zv3TCSvod2f09Bx7ebowLVRzyJe4UG+0OuM10Sk9dXRXL+viizyyPp1Ie2+FN32i",
        "KVTG9jVd21kWUYxT7eKuqH78Jt5Ezmsqs4ArND5qM3B2BWQ9GiyOcOl6NfyA4+RH",
        "wv8eYRJkkjv5q7R675U+EWLe7ktpmboOgl/I5hV1Oj/SQ3F90RqUcLrRz9XTsRKl",
        "nKY2KG/2Q3ZYabf9TpZ/DeHNLus5n4STzFmukQIBIwKCAQEAqF+Nx0TGlCq7P/3Y",
        "GnjAYQr0BAslEoco6KQxkhHDmaaQ0hT8KKlMNlEjGw5Og1TS8UhMRhuCkwsleapF",
        "pksxsZRksc2PJGvVNHNsp4EuyKnz+XvFeJ7NAZheKtoD5dKGk4GrJLhwebf04GyD",
        "30DfxR+nABRAq+nopYBxqFAYSa+Eis0KSd2Gm5w2uuaGBqM1Nqw/EcS41aIFGAvL",
        "gSsAP6ot2W9trWQWGkVvmprFQ64LQ5xwJHf74Ig+t2XjIQ6dkJH6DQjU1nUMMklq",
        "60WagwKBgQDcuFx2GgxbED4Ruv7S/R5ysZuaVpw03S0rKcC3k8lE5xCmrM0E1Q6Z",
        "U2h52ZO4WmXQuTCMh8PIsWKLg7BzacTWd91xGKWE3tD3wXK334fRwVa3ARKgaaH6",
        "Rs1h+a0U8js5T//mf/NYYPKbltWrtXTcuwFt6XG2RWDzn1sPbf8h4wKBgQDJB5m7",
        "ZWVY8+lE2h4QEvql6/YSRTYaYM788FvJDLfh1RS1u0NMu5mOo+0JAKj0JlLzBTsD",
        "drktAHDsAtp0wqH8v2/mZnLYBmK35SwjQ4YNecvLQsIEtmD0USPWKrm1kGdwqohL",
        "q90AJB5HSjBC5Q5vUZVij32WKuSbU+z/t3TH+wKBgBLrOyAQ3HzVgam/ki9XhkRY",
        "XctmgmruYvUSNRcMqtoFLVAdcKikjDkHJjZUemBCQz3GuwS7LgnjUZbuB89g1luG",
        "nfPASLOeEelZuWA3uy88dSWhAZi4mNrwIDuZDtXo4IFBXxPB0weTR/61KEHq+2Ng",
        "fHcio1jEHkDEhCXk21qtAoGAROypvJfK+e06CPpTczm1Ba/8mIzCF6wptc7AYjA/",
        "C5mDcYIIchRvKZdJ9HVBPcP/Lr/2+d+P8iwJdX1SNqkhmHwmXZ931QmA7pe3XIwt",
        "9f3feOOwPCFF0BvRxcWBgBRAuOoC2B2q23oZAn/WCE6ImzHqEynh6lfZWdOhtsKO",
        "cHMCgYBmdhIjJnWbqU5oHVQHN7sVCiRAScAUyTqlUCqB/qSpweZfR+aQ72thnx7C",
        "0j+fdgy90is7ARo9Jam6jFtHwa9JXqH+g24Gdxk+smBeUgiZu63ZG/Z70L4okr4K",
        "6BQlL1pZI4zGbG4H34TPraxvJVdVKVSLAXPur1pqgbJzD2nFUg==",
        "-----END RSA PRIVATE KEY-----"
    ],
    "UserName": "root",
    "ServerPort": "node12:22",
    "Command": "hostname"
}'
