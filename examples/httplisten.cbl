       IDENTIFICATION DIVISION.
       PROGRAM-ID. httplisten.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-STATUS     PIC 9(5) COMP.
       01 REQUEST-DATA.
           05 REQ-METHOD   PIC X(10).
           05 REQ-PATH     PIC X(100).
           05 REQ-BODY     PIC X(256).
       01 WS-DONE       PIC 9(5).
       PROCEDURE DIVISION.
       MAIN.
           PERFORM VARYING WS-DONE FROM 1 BY 1 UNTIL WS-DONE > 3
               HTTP-LISTEN PORT 8080
                   MAPPING REQUEST-DATA
                   STATUS WS-STATUS
                   NOT ON EXCEPTION
                       HTTP-RESPOND STATUS 200
                           BODY "Hello from COBOLabunga!"
                           CONTENT-TYPE "text/plain"
                       END-HTTP
                   ON EXCEPTION
                       DISPLAY "Error"
                END-HTTP
            END-PERFORM
           DISPLAY "Server stopped"
           STOP RUN.
