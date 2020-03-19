RESTSERVER=localhost

 # for Cloud Driver Info
curl -X POST "http://$RESTSERVER:1024/spider/driver" -H 'Content-Type: application/json' -d '{"DriverName":"gcp-driver01","ProviderName":"GCP", "DriverLibFileName":"gcp-driver-v1.0.so"}'

 # for Cloud Credential Info
curl -X POST "http://$RESTSERVER:1024/spider/credential" -H 'Content-Type: application/json' -d '{"CredentialName":"gcp-credential01","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"PrivateKey", "Value":"-----BEGIN PRIVATE KEY-----\nMIIEvgIBADANBgkqhkiG9w0BAQEFA+CfeeZhD1gzc7WNeluK6NUWn6\nP2pci4h0mZL45mI8ulScxEKkiu5TkLwXC2iMPNhu6UkT9hRiyQvhoS5lUcOdQLFE\n4LompuFUNELb30JlkX0OsfVVBwKBgQC+pC2TsbS00oh7dwzLhIQZ6vEQji0UGD2N\nSuBd6luGZRt+8//HwR1urSq31QsYvYXmZkBBlFSGtvVWTRnUdRVFf8m1FgSrcPsn\nKE4/MS1qMesRNZ9ehiLDBHYWTRKSD43bDioc/IjwrJqaLNniVh+kCckI/Hmgg4hV\nOZWxl7tXXQKBgADrpWBTvyzekeDfIopp9TkF1F+iNXuQbRkI65ElRjsaXHMtMJUH\nzHEMBNnvR5C7aIzBdE8wQX1Idr3n87EO02KM0nE8sSNkGDGJsA71XBooukJtY4Xi\ncP00rMmc7Ly6SdEkVnr8kqOl7kiDmSVcgNTsAiLCHw2agzDFneWUHAm5AoGBALLc\nOAKctGz+JZyoqjF7Z7ElUvx0V+jFgWJBwNV8HliuHajzZaPVFDcVcsG8uMeCcNEk\nV97vOoqVtwI8HiLNoqJs7SLfwIvU2V34m8j/65r5sJCZ3acCdDTBx8TOlMDCpRXD\naVF+wUAEwJwrvlRy9wahQ6MRtU8aeNt0xnQzZknlAoGBAMCvEWq/mhE1C4TjPoSy\nj/hJejG6430r/xL3Hdj+rIg59m9xFQbfMEA0FIUUhmRZx6KNjKZsXUZjcUaFRnen\nUSIYLKrRT61dgmnc7EQ6gUnjjBeCEwc6TZnlTAiorkeSal++nhfPEHii8Cqe8ter\nNSzr3u0xEHxZkYPSwhZEX5ex\n-----END PRIVATE KEY-----\n"},{"Key":"ProjectID", "Value":"mcloud-barista"}, {"Key":"ClientEmail", "Value":"67811253-compute@developer.gserviceaccount.com"}]}'


 # for Cloud Region Info
curl -X POST "http://$RESTSERVER:1024/spider/region" -H 'Content-Type: application/json' -d '{"RegionName":"gcp-region01","ProviderName":"GCP", "KeyValueInfoList": [{"Key":"Region", "Value":"asia-northeast1"},{"Key":"Zone", "Value":"asia-northeast1-b"}]}'

 # for Cloud Connection Config Info
curl -X POST "http://$RESTSERVER:1024/spider/connectionconfig" -H 'Content-Type: application/json' -d '{"ConfigName":"gcp-config01","ProviderName":"GCP", "DriverName":"gcp-driver01", "CredentialName":"gcp-credential01", "RegionName":"gcp-region01"}'
