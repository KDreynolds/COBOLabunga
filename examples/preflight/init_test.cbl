       IDENTIFICATION DIVISION.
       PROGRAM-ID. init.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-STR           PIC X(10) VALUE "ABCDEFGHIJ".
       01 WS-NUM           PIC 9(5) COMP VALUE 42.
       PROCEDURE DIVISION.
       MAIN.
           INITIALIZE WS-STR
           IF WS-STR = "          "
               DISPLAY 1
           ELSE
               DISPLAY 0
           END-IF
           INITIALIZE WS-NUM
           IF WS-NUM = 0
               DISPLAY 1
           ELSE
               DISPLAY 0
           END-IF
           STOP RUN.
