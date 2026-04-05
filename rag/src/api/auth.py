"""JWT authentication for RAG API."""

from fastapi import Header, HTTPException, Security, status
from fastapi.security import HTTPBearer, HTTPAuthorizationCredentials
from jose import jwt, JWTError
from ..config.settings import get_settings

security = HTTPBearer(auto_error=False)


async def get_current_tenant_id(
    credentials: HTTPAuthorizationCredentials | None = Security(security),
    x_tenant_id: str | None = Header(default=None),
) -> str:
    settings = get_settings()

    if credentials is None:
        if x_tenant_id:
            return x_tenant_id
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail="Missing authentication credentials",
        )

    try:
        token = credentials.credentials
        payload = jwt.decode(
            token,
            settings.jwt_secret,
            algorithms=[settings.jwt_algorithm]
        )

        tenant_id = payload.get("tenant_id")
        if not tenant_id:
            raise HTTPException(
                status_code=status.HTTP_401_UNAUTHORIZED,
                detail="Token missing tenant_id claim"
            )

        return tenant_id

    except JWTError as e:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail=f"Invalid token: {str(e)}"
        )
