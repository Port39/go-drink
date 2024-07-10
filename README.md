# Go get something to drink!

Once mature enough, this will be the backend of our solution to manage the mate & drinks supply of our Hackerspace. As 
such, there are two principles that must be prioritized when making design decisions that affect the way users 
interact with the system:

 * **It must be easy.**
    
    If you just want to drink some mate, you usually don't want to RTFM first. Therefore, using the whole system must be 
    intuitive, and not require another person explaining it first. 

 * **It must be fast.**

    The purpose of this system lays in enabling users to grab a cold drink, and not in wasting their time. So the steps 
    that are necessary to make a transaction must be reduced to a minimum. By giving users a variety of authentication 
    options, they can choose for themselves what gives "good enough" security and a fast user experience. 

## External Dependencies
This backend depends on several other services, that should be configured alongside with this application. Not all are 
strictly required, but some functions might not work if a service is configured improperly. 

### Postgres
go-drink uses Postgresql as a database backend and fully depends on it. If a database backend is not configured, or if 
it is unreachable during the application startup, go-drink terminates instantly.

| Environment Variable | Example Value | Notes                                                         |
| --- | --- |---------------------------------------------------------------|
| GODRINK_DB | `postgresql://godrink:changeme@db:5432/godrink?sslmode=disable` | A connection string describing of the database can be reached | 

### SMTP / Mailing
An SMTP server can be configured, so that the application can send out emails to users, for example if a password reset 
is requested. If the SMTP configuration is omitted, or contains invalid data, a warning is logged, but the application 
functions otherwise normally.

| Environment Variable | Example Value                  | Notes                                                                                                              |
| --- |--------------------------------|--------------------------------------------------------------------------------------------------------------------|
| GODRINK_SMTPHOST | `yourmailhost.example:465`     | The address of the SMTP Server, in the format <host>:<port>                                                        |
| GODRINK_SMTPUSER | `godrink@yourmailhost.example` | The username that should be used to authenticate to the server                                                     |
| GODRINK_SMTPPASS | `changeme` | The corresponding password for the user.                                                                           |
| GODRINK_SMTPFROM | `godrink@yourmailhost.example` | The address that is used in the `From` header header when sending out emails. If omitted, the value of `GODRINK_SMTPUSER` is used. |

**A note on TLS:** Currently, the application expects that it can open a TLS encrypted connection to the target port. 
STARTTLS or plaintext communication is not supported at the moment. 