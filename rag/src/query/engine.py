"""RAG query engine for retrieving relevant documents and generating answers."""

from typing import List, Dict, Any, Optional
from langchain_core.prompts import ChatPromptTemplate
from langchain_core.runnables import RunnablePassthrough
from langchain_core.output_parsers import StrOutputParser
from langchain_core.language_models import BaseChatModel

from ..vectorstore.db9_vectorstore import DB9VectorStore
from ..config.settings import get_settings


class RAGQueryEngine:
    """Query engine for RAG system."""

    def __init__(
        self,
        vectorstore: DB9VectorStore,
        llm: Optional[BaseChatModel] = None,
    ):
        self._vectorstore = vectorstore
        self._settings = get_settings()

        if llm:
            self._llm = llm
        elif self._settings.provider == "zhipuai":
            from langchain_community.chat_models import ChatZhipuAI

            self._llm = ChatZhipuAI(
                model=self._settings.zhipuai_llm_model,
                temperature=self._settings.llm_temperature,
                api_key=self._settings.zhipuai_api_key,
            )
        elif self._settings.provider == "openai":
            from langchain_openai import ChatOpenAI

            self._llm = ChatOpenAI(
                model=self._settings.llm_model,
                temperature=self._settings.llm_temperature,
            )
        else:
            self._llm = None
        self._chain = self._create_chain() if self._llm else None

    def _create_chain(self):
        """Create the RAG chain."""
        template = """Answer the question based only on the following context:

{context}

Question: {question}

Answer:"""

        prompt = ChatPromptTemplate.from_template(template)

        return (
            {
                "context": self._vectorstore.as_retriever(
                    search_kwargs={"k": self._settings.default_k}
                ) | self._format_docs,
                "question": RunnablePassthrough(),
            }
            | prompt
            | self._llm
            | StrOutputParser()
        )

    def _format_docs(self, docs: List[Dict[str, Any]]) -> str:
        """Format retrieved documents."""
        return "\n\n".join([
            f"[{i+1}] {doc.page_content}"
            for i, doc in enumerate(docs)
        ])

    def _generate_local_answer(self, question: str, docs: List[Dict[str, Any]]) -> str:
        if not docs:
            return f"未检索到与问题“{question}”相关的本地上下文。"

        excerpts = []
        for index, doc in enumerate(docs[:3], start=1):
            content = " ".join(doc.page_content.split())
            excerpts.append(f"[{index}] {content[:240]}")

        return "\n\n".join(excerpts)

    def query(
        self,
        question: str,
        k: Optional[int] = None,
        filter: Optional[Dict[str, Any]] = None,
    ) -> Dict[str, Any]:
        """Query the RAG system."""
        import time
        start_time = time.time()

        # Retrieve relevant chunks
        k = k or self._settings.default_k
        results = self._vectorstore.similarity_search(
            query=question,
            k=k,
            filter=filter,
        )

        if self._chain is None:
            response = self._generate_local_answer(question, results)
        else:
            response = self._chain.invoke(question)

        latency = (time.time() - start_time) * 1000

        return {
            "question": question,
            "answer": response,
            "sources": [doc.metadata for doc in results],
            "latency_ms": latency,
        }
