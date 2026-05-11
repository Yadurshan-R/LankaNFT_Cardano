import { Request, Response, NextFunction } from "express";

/**
 * Shared secret middleware
 * Every request from Go backend must include this header:
 * X-Service-Secret: <SHARED_SECRET from .env>
 * If it does not match, we reject the request immediately
 * This prevents anyone else from calling this service directly
 */
export const validateSecret = (
  req: Request,
  res: Response,
  next: NextFunction
): void => {
  const secret = req.headers["x-service-secret"];

  if (!secret || secret !== process.env.SHARED_SECRET) {
    res.status(401).json({
      error: "Unauthorized — invalid service secret",
    });
    return;
  }

  // Secret matches — continue to the route handler
  next();
};