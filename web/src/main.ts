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

type ArithmeticOperation = 'soma' | 'subtracao' | 'multiplicacao' | 'divisao';

const form = document.querySelector<HTMLFormElement>('#meu-formulario');

const form1 = document.querySelector<HTMLInputElement>('#campo1');
const form2 = document.querySelector<HTMLInputElement>('#campo2');
const operacao = document.querySelector<HTMLSelectElement>('#operacao');

form.addEventListener('submit', printInputValues);

interface CalculateRequest {
  firstValue: number;
  secondValue: number;
  operacao: ArithmeticOperation;
}

async function printInputValues(event) {
  event.preventDefault();

  await tracer.startActiveSpan('calc-request', async (span) => {
    const request: CalculateRequest = {
      firstValue: form1.value,
      secondValue: form2.value,
      operacao: operacao.value,
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
      console.log(data);
      span.addEvent('request-completed');
    } catch (ex: Error) {
      console.log(ex);
      span.recordException(ex);
    } finally {
      span.end();
    }
  });
}
