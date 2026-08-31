"use strict";
const form = document.querySelector('#meu-formulario');
form.addEventListener('submit', printInputValues);
function printInputValues(event) {
    const form1 = document.querySelector('#campo1');
    const form2 = document.querySelector('#campo2');
    const operacao = document.querySelector('#operacao');
    console.log(form1.value);
    console.log(form2.value);
    console.log(operacao.value);
    event.preventDefault();
}
