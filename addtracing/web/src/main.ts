import { WebTracerProvider } from '@opentelemetry/sdk-trace-web';
import { ZoneContextManager } from '@opentelemetry/context-zone';
import {
  SimpleSpanProcessor,
  ConsoleSpanExporter,
} from '@opentelemetry/sdk-trace-base';
import { registerInstrumentations } from '@opentelemetry/instrumentation';
import { FetchInstrumentation } from '@opentelemetry/instrumentation-fetch';

import { trace } from '@opentelemetry/api';
const tracer = trace.getTracer('calc-tracer');

const provider = new WebTracerProvider({
  spanProcessors: [new SimpleSpanProcessor(new ConsoleSpanExporter())],
});

provider.register({
  contextManager: new ZoneContextManager(),
});

registerInstrumentations({
  instrumentations: [
    new FetchInstrumentation({
      propagateTraceHeaderCorsUrls: ['http://localhost:8080/calculate'],
    }),
  ],
});

type ArithmeticOperation = 'sum' | 'sub' | 'mult' | 'div';

const form = document.querySelector<HTMLFormElement>('#meu-formulario');
const input = document.querySelector<HTMLInputElement>('#campo1');
const operacao = document.querySelector<HTMLSelectElement>('#operacao');
const respostaDiv = document.querySelector<HTMLDivElement>('#resposta');
const resultadoConteudo = document.querySelector<HTMLDivElement>(
  '#resultado-conteudo',
);

form.addEventListener('submit', printInputValues);

interface CalculateRequest {
  firstValue: number;
  secondValue: number;
  operacao: ArithmeticOperation;
}

async function printInputValues(event) {
  event.preventDefault();

  await tracer.startActiveSpan('calc-request', async (span) => {
    const operands = input.value
      .split(',')
      .map((item) => item.trim())
      .filter((item) => item !== '')
      .map(Number)
      .join(',');

    const request: CalculateRequest = {
      operands,
      operator: operacao.value as ArithmeticOperation,
    };

    try {
      const response = await fetch('http://localhost:8080/calculate', {
        method: 'POST',
        headers: {
          'Content-Type': 'Application/json',
        },
        body: JSON.stringify(request),
      });
      const data = await response.json();

      respostaDiv.classList.add('active');
      resultadoConteudo.className = 'result-value';
      resultadoConteudo.textContent =
        data.result !== undefined ? data.result : JSON.stringify(data);

      span.addEvent('request-completed');
    } catch (ex: Error) {
      console.log(ex);
      //TODO show something
      span.recordException(ex);
    } finally {
      span.end();
    }
  });
}
