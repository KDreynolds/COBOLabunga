       IDENTIFICATION DIVISION.
       PROGRAM-ID. httpmap.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 RESPONSE-DATA.
          05 USER-ID    PIC 9(5).
          05 ID         PIC 9(5).
          05 TITLE      PIC X(100).
          05 BODY       PIC X(200).
       01 WS-STATUS     PIC 9(5) COMP.
       PROCEDURE DIVISION.
       MAIN.
           HTTP-GET "https://jsonplaceholder.typicode.com/posts/1"
               MAPPING RESPONSE-DATA
               STATUS WS-STATUS
           END-HTTP
           DISPLAY "Status: " WS-STATUS
           DISPLAY "Title: " TITLE
           DISPLAY "User: " USER-ID
           DISPLAY "ID: " ID
           STOP RUN.
