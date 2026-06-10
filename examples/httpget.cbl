       IDENTIFICATION DIVISION.
       PROGRAM-ID. httpget.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-RESPONSE   PIC X(200).
       01 WS-STATUS     PIC 9(5) COMP.
       PROCEDURE DIVISION.
       MAIN.
           HTTP-GET "https://httpbin.org/get"
               GIVING WS-RESPONSE
               STATUS WS-STATUS
           END-HTTP
           DISPLAY "Status: " WS-STATUS
           DISPLAY WS-RESPONSE
           STOP RUN.
