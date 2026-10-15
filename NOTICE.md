# OPEN SOURCE SOFTWARE NOTICE

Please note we provide an open source software notice along with this product (in the following just “this product”). The open source software licenses are granted by the respective right holders. And the open source licenses prevail all other license information with regard to the respective open source software contained in the product, including but not limited to End User Software Licensing Agreement. This notice is provided on behalf of Cloud-Barista Community.


#### Warranty Disclaimer

The open source software in this product is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY, without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the applicable licenses for more details.


#### Copyright Notice

Software : Cloud-Barista CB-Spider

Copyright notice : <BR>
 Copyright (C) 2019 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2020 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2021 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2022 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2023 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2024 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2025 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>
 Copyright (C) 2026 - Cloud-Barista Community (https://cloud-barista.github.io) <BR>

License : Apache License 2.0 <BR>
 [Apache License 2.0 original text](https://github.com/cloud-barista/cb-spider/blob/master/LICENSE)


#### Third-Party Software Notices

This product bundles a number of third-party open source components. Per Section 4(d) of the Apache License, Version 2.0, the attribution notices contained in the NOTICE files of the following Apache-2.0-licensed dependencies are reproduced below.

**Go module dependencies (Apache License 2.0) with their own NOTICE file**

- github.com/aws/aws-sdk-go <BR>
  ```
  AWS SDK for Go
  Copyright 2015 Amazon.com, Inc. or its affiliates. All Rights Reserved.
  Copyright 2014-2015 Stripe, Inc.
  ```

- github.com/oracle/oci-go-sdk/v65 <BR>
  ```
  Copyright (c) 2016, 2018, 2026, Oracle and/or its affiliates.
  ```

- github.com/minio/minio-go/v7 <BR>
  ```
  MinIO Cloud Storage, (C) 2014-2020 MinIO, Inc.

  This product includes software developed at MinIO, Inc.
  (https://min.io/).

  The MinIO project contains unmodified/modified subcomponents too with
  separate copyright notices and license terms. Your use of the source
  code for these subcomponents is subject to the terms and conditions
  of Apache License Version 2.0
  ```

- github.com/go-openapi/jsonpointer, github.com/go-openapi/jsonreference <BR>
  ```
  Copyright 2015-2025 go-swagger maintainers

  This software library includes software developed by the go-swagger and
  go-openapi maintainers ("go-swagger maintainers"), copied from, derived
  from, and inspired by the original software authored on 25-02-2013 by
  sigu-399 (https://github.com/sigu-399), Copyright 2013 sigu-399.

  Licensed under the Apache License, Version 2.0.
  ```

- gopkg.in/yaml.v2, gopkg.in/yaml.v3 <BR>
  ```
  Copyright 2011-2016 Canonical Ltd.

  Licensed under the Apache License, Version 2.0.
  ```

**Bundled frontend libraries (MIT License)**

The admin web console and SpiderWatch web UI bundle the following third-party JavaScript/CSS libraries as static assets (not managed via go.mod):

- `api-runtime/rest-runtime/admin-web/static/js/xterm.js`, `api-runtime/rest-runtime/admin-web/static/css/xterm.css` — **xterm.js** <BR>
  Copyright (c) 2017-2019, The xterm.js authors (https://github.com/xtermjs/xterm.js) <BR>
  Copyright (c) 2014-2016, SourceLair Private Company (https://www.sourcelair.com) <BR>
  Copyright (c) 2012-2013, Christopher Jeffrey (https://github.com/chjj/) <BR>
  License: MIT — full text at `api-runtime/rest-runtime/admin-web/THIRD-PARTY-LICENSE-xterm.js.txt`

- `spiderwatch/web/static/js/marked.min.js` — **marked** <BR>
  Copyright (c) 2011-2025, Christopher Jeffrey (https://github.com/markedjs/marked) <BR>
  License: MIT

**Go module dependencies (Mozilla Public License 2.0)**

The following dependencies are used unmodified and are covered by MPL-2.0 Section 3.3 ("Larger Work"), which permits combination with software under other licenses such as Apache-2.0:

- github.com/go-sql-driver/mysql
- github.com/hashicorp/go-cleanhttp
- github.com/hashicorp/go-retryablehttp
- github.com/hashicorp/go-version
- github.com/bramvdbogaerde/go-scp

Should any of these files be modified and redistributed in the future, those modified files must remain licensed under MPL-2.0 and their source made available, per the terms of that license.

**Full dependency list**

CB-Spider depends on a large number of additional open source Go modules licensed under Apache-2.0, MIT, BSD-2-Clause, BSD-3-Clause, and ISC. The authoritative, versioned list of all such dependencies is maintained in [go.mod](https://github.com/cloud-barista/cb-spider/blob/master/go.mod) and [go.sum](https://github.com/cloud-barista/cb-spider/blob/master/go.sum); their respective license texts are distributed with each module and are not reproduced individually here.
