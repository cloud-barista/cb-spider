openssl s_client -showcerts -connect localhost:10241 </dev/null 2>/dev/null | openssl x509 -outform PEM > cert.pem
