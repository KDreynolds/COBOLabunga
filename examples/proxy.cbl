       IDENTIFICATION DIVISION.
       PROGRAM-ID. proxy.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-STATUS       PIC 9(5) COMP.
       01 WS-REMOTE-STAT  PIC 9(5) COMP.
       01 WS-RESPONSE     PIC X(3000).
       01 WS-DONE         PIC 9(5).
       01 REQUEST-DATA.
           05 REQ-METHOD   PIC X(10).
           05 REQ-PATH     PIC X(100).
           05 REQ-BODY     PIC X(500).
       01 RESPONSE-HEADERS.
           05 X-PROXY     PIC X(20).
           05 X-BACKEND   PIC X(20).
       PROCEDURE DIVISION.
       MAIN.
           MOVE "COBOLabunga" TO X-PROXY
           MOVE "httpbin.org" TO X-BACKEND
           DISPLAY "Proxy server starting on port 8080..."
           PERFORM VARYING WS-DONE FROM 1 BY 1 UNTIL WS-DONE > 5
               HTTP-LISTEN PORT 8080
                   MAPPING REQUEST-DATA
                   STATUS WS-STATUS
                   NOT ON EXCEPTION
                       DISPLAY "--> " REQ-METHOD " " REQ-PATH
                       HTTP-GET "https://httpbin.org/get"
                           GIVING WS-RESPONSE
                           STATUS WS-REMOTE-STAT
                       END-HTTP
                       DISPLAY "<-- " WS-REMOTE-STAT
                       HTTP-RESPOND STATUS 200
                           BODY WS-RESPONSE
                           CONTENT-TYPE "application/json"
                           HEADERS RESPONSE-HEADERS
                       END-HTTP
                   ON EXCEPTION
                       DISPLAY "Error processing request"
                       HTTP-RESPOND STATUS 500
                           BODY "Internal Server Error"
                           CONTENT-TYPE "text/plain"
                       END-HTTP
               END-HTTP
           END-PERFORM
           DISPLAY "Server stopped"
           STOP RUN.
