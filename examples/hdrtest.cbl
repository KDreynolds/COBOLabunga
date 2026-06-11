       IDENTIFICATION DIVISION.
       PROGRAM-ID. hdrtest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-STATUS     PIC 9(5) COMP.
       01 RESPONSE-HEADERS.
           05 X-CUSTOM   PIC X(30).
           05 CACHE-CONTROL  PIC X(30).
       01 WS-DONE       PIC 9(5).
       PROCEDURE DIVISION.
       MAIN.
           MOVE "my-value" TO X-CUSTOM
           MOVE "no-cache" TO CACHE-CONTROL
           PERFORM VARYING WS-DONE FROM 1 BY 1 UNTIL WS-DONE > 1
               HTTP-LISTEN PORT 8081
                   STATUS WS-STATUS
                   NOT ON EXCEPTION
                       HTTP-RESPOND STATUS 200
                           BODY "Headers test"
                           CONTENT-TYPE "text/plain"
                           HEADERS RESPONSE-HEADERS
                       END-HTTP
                   ON EXCEPTION
                       DISPLAY "Error"
               END-HTTP
           END-PERFORM
           DISPLAY "Server stopped"
           STOP RUN.
