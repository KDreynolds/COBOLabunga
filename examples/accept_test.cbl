       IDENTIFICATION DIVISION.
       PROGRAM-ID. acceptest.
       DATA DIVISION.
       WORKING-STORAGE SECTION.
       01 WS-NAME PIC X(20).
       01 WS-AGE  PIC 9(4).
       PROCEDURE DIVISION.
       MAIN.
           DISPLAY "Enter name:"
           ACCEPT WS-NAME
           DISPLAY "Enter age:"
           ACCEPT WS-AGE
           DISPLAY "Name:"
           DISPLAY WS-NAME
           DISPLAY "Age:"
           DISPLAY WS-AGE
           STOP RUN.
