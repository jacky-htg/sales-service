# sales-service
Sales service using golang grpc and postgresql. This service is part of inventory microservices. 

- The service is part of ERP microservices.
- The service will be call in local network.
- Using grpc insecure connection

## Get Started
- git clone git@github.com:jacky-htg/sales-service.git
- make init
- cp .env.example .env (and edit with your environment)
- make migrate
- make seed
- make server
- You can test the service using grpc client like wombat or grpcurl

## Features
- [X] Salesman
- [X] Customers
- [X] Sales
- [X] Sales Returns

## How To Contribute
- Give star or clone and fork the repository
- Report the bug
- Submit issue for request of enhancement
- Pull Request for fixing bug or enhancement module

## License
[The license of application is GPL-3.0](./LICENSE), You can use this apllication for commercial use, distribution or modification. But there is no liability and warranty. Please read the license details carefully.

## Link Repository
- [API Gateway for ERP](https://github.com/jacky-htg/erp-gateway-service)
- [User Service](https://github.com/jacky-htg/user-service)
- [Purchase Service](https://github.com/jacky-htg/purchase-service)
- [Inventory Service](https://github.com/jacky-htg/inventory-service)
- [General Ledger Service](https://github.com/jacky-htg/ledger-service)