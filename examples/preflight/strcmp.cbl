       IDENTIFICATION DIVISION.
       PROGRAM-ID. strcmp.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-INPUT        PIC X(10) VALUE "HELLO".
       PROCEDURE DIVISION.
       MAIN.
           IF WS-INPUT = "HELLO"
               DISPLAY 1
           ELSE
               DISPLAY 0
           END-IF
           IF WS-INPUT = "WORLD"
               DISPLAY 0
           ELSE
               DISPLAY 1
           END-IF
           STOP RUN.
